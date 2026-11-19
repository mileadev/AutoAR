package brain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	ollamaLocalURL = "http://127.0.0.1:11434"
	ollamaCloudURL = "https://ollama.com"
	ollamaMaxResponseBytes = 4 << 20
)

// ollamaModel supports independent model choices for local and cloud inference.
func ollamaModel(provider string) string {
	if provider == "ollama-cloud" {
		if model := strings.TrimSpace(os.Getenv("OLLAMA_CLOUD_MODEL")); model != "" {
			return model
		}
	}
	return strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
}

func ollamaEndpoint(provider string) (string, error) {
	base := ollamaLocalURL
	if provider == "ollama-cloud" {
		base = ollamaCloudURL // Fixed host: never send the cloud bearer key to a custom URL.
	} else if provider == "ollama" {
		if value := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); value != "" {
			base = value
		}
	} else {
		return "", fmt.Errorf("unsupported Ollama provider %q", provider)
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid Ollama base URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("OLLAMA_BASE_URL must be an http(s) origin without credentials, path, query, or fragment")
	}
	return strings.TrimRight(parsed.String(), "/") + "/api/chat", nil
}

// callOllama uses Ollama's non-streaming native API for both modes.
// Explicitly selected Ollama providers never fall back to third-party AI services.
func callOllama(messages []Message, provider string) (string, error) {
	model := ollamaModel(provider)
	if model == "" {
		return "", fmt.Errorf("set OLLAMA_MODEL (or OLLAMA_CLOUD_MODEL for cloud) before using %s", provider)
	}
	endpoint, err := ollamaEndpoint(provider)
	if err != nil {
		return "", err
	}
	apiKey := ""
	if provider == "ollama-cloud" {
		apiKey = strings.TrimSpace(os.Getenv("OLLAMA_API_KEY"))
		if apiKey == "" {
			return "", fmt.Errorf("OLLAMA_API_KEY is required for direct Ollama Cloud access")
		}
	}
	client := &http.Client{
		Timeout: 180 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Prevent provider credential forwarding.
		},
	}
	return callOllamaEndpoint(messages, endpoint, model, apiKey, client)
}

// callOllamaEndpoint is separate to allow deterministic, offline HTTP contract tests.
func callOllamaEndpoint(messages []Message, endpoint, model, apiKey string, client *http.Client) (string, error) {
	payload, err := json.Marshal(struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
		Stream   bool      `json:"stream"`
	}{Model: model, Messages: messages, Stream: false})
	if err != nil {
		return "", fmt.Errorf("encode Ollama request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, ollamaMaxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read Ollama response: %w", err)
	}
	if len(body) > ollamaMaxResponseBytes {
		return "", fmt.Errorf("Ollama response exceeds %d bytes", ollamaMaxResponseBytes)
	}
	if resp.StatusCode != http.StatusOK {
		// Do not echo the response body: it may contain prompts or secrets.
		return "", fmt.Errorf("Ollama API returned HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Message Message `json:"message"`
		Error   string  `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("invalid Ollama JSON response: %w", err)
	}
	if decoded.Error != "" {
		return "", fmt.Errorf("Ollama inference returned an error")
	}
	if strings.TrimSpace(decoded.Message.Content) == "" {
		return "", fmt.Errorf("Ollama returned no assistant content; check model output and context length")
	}
	return decoded.Message.Content, nil
}
