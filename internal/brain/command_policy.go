package brain

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// An LLM is an untrusted command proposer. CLI execution is opt-in and
// restricted to approved subcommands and domains, not arbitrary binaries.
func validateAIAgentCommand(action string, parts []string) error {
	if os.Getenv("AUTOAR_AI_EXECUTE_COMMANDS") != "true" {
		return fmt.Errorf("AI execution disabled")
	}
	if action != "run_command" {
		return fmt.Errorf("model-generated shell commands are prohibited")
	}
	if len(parts) < 2 || len(parts) > 10 || parts[0] != "autoar" {
		return fmt.Errorf("invalid AutoAR command")
	}
	if len(parts) == 3 && parts[1] == "status" && parts[2] == "--json" {
		return nil
	}
	if len(parts) < 4 {
		return fmt.Errorf("expected a scoped AutoAR command")
	}

	allowed := map[string]bool{
		"subdomains get": true, "livehosts get": true,
		"ports scan": true, "nuclei run": true,
		"tech detect": true, "urls collect": true,
		"cnames get": true, "dns takeover": true,
		"gf scan": true, "reflection scan": true,
		"ffuf fuzz": true, "zerodays scan": true,
		"domain run": true, "recon run": true,
		"lite run": true, "subdomain run": true,
	}
	if !allowed[parts[1]+" "+parts[2]] {
		return fmt.Errorf("AutoAR subcommand is not authorized for the AI agent")
	}

	var target string
	for i := 3; i < len(parts); {
		if i+1 >= len(parts) {
			return fmt.Errorf("missing value for command flag")
		}
		flag, value := parts[i], parts[i+1]
		if len(value) > 253 || strings.ContainsAny(value, "\x00\n\r\t /\\\"';&|$`<>") {
			return fmt.Errorf("invalid command argument")
		}
		switch flag {
		case "-d", "-s":
			if target != "" { return fmt.Errorf("only one target allowed") }
			target = strings.ToLower(strings.TrimSuffix(value, "."))
		case "-t":
			threads, err := strconv.Atoi(value)
			if err != nil || threads < 1 || threads > 16 {
				return fmt.Errorf("AI thread count exceeds safe limits")
			}
		default:
			return fmt.Errorf("unapproved AI command flag")
		}
		i += 2
	}
	if target == "" { return fmt.Errorf("AI scan target is required") }
	if net.ParseIP(target) != nil { return fmt.Errorf("IP targets are not authorized by domain scope") }
	if strings.Contains(target, "..") || strings.HasPrefix(target, "-") ||
		strings.HasSuffix(target, "-") || strings.Contains(target, ":") {
		return fmt.Errorf("invalid target hostname")
	}
	for _, c := range target {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
			return fmt.Errorf("invalid hostname character")
		}
	}
	for _, raw := range strings.Split(os.Getenv("AUTOAR_AI_ALLOWED_DOMAINS"), ",") {
		scope := strings.ToLower(strings.Trim(strings.TrimSpace(raw), "."))
		if scope == "" || strings.ContainsAny(scope, " /:\\") { continue }
		if target == scope || strings.HasSuffix(target, "."+scope) {
			return nil
		}
	}
	return fmt.Errorf("target outside AUTOAR_AI_ALLOWED_DOMAINS")
}
