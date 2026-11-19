package brain

import (
	"fmt"
	"os"
	"strings"
)

// AI-generated instructions are untrusted data, even when the user asked the agent to scan.
// Opt in to execution separately from inference; shell tools require an additional opt in.
func validateAIAgentCommand(action string, parts []string) error {
	if os.Getenv("AUTOAR_AI_EXECUTE_COMMANDS") != "true" {
		return fmt.Errorf("AI command execution disabled (set AUTOAR_AI_EXECUTE_COMMANDS=true only in an isolated, authorized environment)")
	}
	if len(parts) == 0 {
		return fmt.Errorf("empty AI command")
	}
	if len(parts) > 64 {
		return fmt.Errorf("AI command has too many arguments")
	}
	for _, part := range parts {
		if len(part) > 4096 || strings.ContainsAny(part, "\x00\n\r") {
			return fmt.Errorf("invalid AI command argument")
		}
	}
	switch action {
	case "run_command":
		// Never allow a model to choose an arbitrary program through run_command.
		if parts[0] != "autoar" {
			return fmt.Errorf("run_command only supports the autoar executable")
		}
	case "run_shell":
		if os.Getenv("AUTOAR_AI_ALLOW_SHELL") != "true" {
			return fmt.Errorf("AI shell commands disabled (AUTOAR_AI_ALLOW_SHELL is not true)")
		}
		// Shell interpreters and arbitrary binaries are deliberately not supported.
		switch parts[0] {
		case "curl", "nmap":
			// Explicit opt in is required; use only against targets in approved scope.
		default:
			return fmt.Errorf("AI shell command executable is not allowlisted")
		}
	default:
		return fmt.Errorf("unsupported AI command action %q", action)
	}
	return nil
}
