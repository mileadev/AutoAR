package brain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOllamaEndpointValidation(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://100.108.101.105:11434")
	endpoint, err := ollamaEndpoint("ollama")
	if err != nil || endpoint != "http://100.108.101.105:11434/api/chat" {
		t.Fatalf("unexpected endpoint %q: %v", endpoint, err)
	}
	for _, invalid := range []string{
		"file:///etc/passwd",
		"http://user:password@localhost:11434",
		"http://localhost:11434/suspicious",
		"http://localhost:11434?key=secret",
	} {
		t.Setenv("OLLAMA_BASE_URL", invalid)
		if _, err := ollamaEndpoint("ollama"); err == nil {
			t.Errorf("accepted invalid endpoint %q", invalid)
		}
	}
	t.Setenv("OLLAMA_BASE_URL", "http://evil.example")
	if endpoint, err := ollamaEndpoint("ollama-cloud"); err != nil || endpoint != "https://ollama.com/api/chat" {
		t.Fatalf("cloud endpoint must be pinned: %q, %v", endpoint, err)
	}
}

func TestOllamaChatContractAndHeaders(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Errorf("unexpected HTTP request %s %s", r.Method, r.URL.Path)
		}
		var request struct {
			Model string `json:"model"`
			Stream *bool `json:"stream"`
			Messages []Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode: %v", err)
		}
		if request.Model != "qwen3:8b" || request.Stream == nil || *request.Stream || len(request.Messages) != 2 {
			t.Errorf("bad Ollama payload: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"verified"}}`))
	}))
	defer server.Close()
	messages := []Message{{Role:"system",Content:"safe instruction"},{Role:"user",Content:"hello"}}
	received, err := callOllamaEndpoint(messages, server.URL+"/api/chat", "qwen3:8b", "", server.Client())
	if err != nil || received != "verified" || gotAuth != "" {
		t.Fatalf("local response %q, auth %q, error %v", received, gotAuth, err)
	}
	received, err = callOllamaEndpoint(messages, server.URL+"/api/chat", "qwen3:8b", "test-key", server.Client())
	if err != nil || received != "verified" || gotAuth != "Bearer test-key" {
		t.Fatalf("cloud auth mismatch: received %q auth %q error %v", received, gotAuth, err)
	}
}

func TestOllamaRejectsErrorBodyDisclosureAndOversize(t *testing.T) {
	for _, tc := range []struct{status int; body string}{
		{401, "secret=do-not-echo"},
		{200, `{"message":{"content":""}}`},
		{200, strings.Repeat("x", ollamaMaxResponseBytes+1)},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		_, err := callOllamaEndpoint(nil, server.URL, "model", "", &http.Client{Timeout:time.Second * 10})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "do-not-echo") {
			t.Errorf("bad error handling for status %d: %v", tc.status, err)
		}
	}
}

func TestExplicitOllamaModeNeverFallsBack(t *testing.T) {
	t.Setenv("AI_PROVIDER","ollama")
	t.Setenv("OLLAMA_MODEL","")
	t.Setenv("OPENCODE_API_KEY","legacy-key")
	if hasAnyAIKey() {
		t.Fatal("legacy key must not enable unconfigured Ollama")
	}
	t.Setenv("OLLAMA_MODEL","qwen3:8b")
	if !hasAnyAIKey() {
		t.Fatal("local Ollama should not require any cloud API key")
	}
	t.Setenv("AI_PROVIDER","ollama-cloud")
	t.Setenv("OLLAMA_API_KEY","")
	if hasAnyAIKey() {
		t.Fatal("cloud mode must require an Ollama API key")
	}
	t.Setenv("OLLAMA_API_KEY","cloud-key")
	if !hasAnyAIKey() {
		t.Fatal("cloud mode with model/key should be available")
	}
	_ = os.Getenv("AI_PROVIDER")
}
