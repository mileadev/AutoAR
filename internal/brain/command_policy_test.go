package brain

import (
	"os"
	"strings"
	"testing"
)

func TestAIExecutionDefaultDeny(t *testing.T) {
	t.Setenv("AUTOAR_AI_EXECUTE_COMMANDS", "")
	t.Setenv("AUTOAR_AI_ALLOW_SHELL", "")
	if err := validateAIAgentCommand("run_command", []string{"autoar", "status"}); err == nil {
		t.Fatal("expected execution to be disabled by default")
	}
	if err := validateAIAgentCommand("run_shell", []string{"curl", "https://example.org"}); err == nil {
		t.Fatal("expected shell execution to be disabled by default")
	}
}

func TestAIExecutionRequiresIndependentShellOptIn(t *testing.T) {
	t.Setenv("AUTOAR_AI_EXECUTE_COMMANDS", "true")
	t.Setenv("AUTOAR_AI_ALLOW_SHELL", "")
	if err := validateAIAgentCommand("run_command", []string{"autoar", "status", "--json"}); err != nil {
		t.Fatalf("allowed command rejected: %v", err)
	}
	for _, tc := range [][]string{
		{"bash", "-c", "id"},
		{"/bin/sh", "-c", "id"},
		{"curl", "https://example.org"},
		{"env"},
	} {
		if err := validateAIAgentCommand("run_shell", tc); err == nil {
			t.Fatalf("unapproved shell allowed: %q", tc)
		}
	}
}

func TestAIExecutionAllowlist(t *testing.T) {
	t.Setenv("AUTOAR_AI_EXECUTE_COMMANDS", "true")
	t.Setenv("AUTOAR_AI_ALLOW_SHELL", "true")
	for _, command := range []string{"curl", "nmap"} {
		if err := validateAIAgentCommand("run_shell", []string{command, "--help"}); err != nil {
			t.Fatalf("%s wrongly blocked: %v", command, err)
		}
	}
	if err := validateAIAgentCommand("run_shell", []string{"python3", "-c", "print(1)"}); err == nil {
		t.Fatal("arbitrary executable allowed")
	}
	if err := validateAIAgentCommand("run_command", []string{"curl", "https://example.org"}); err == nil {
		t.Fatal("run_command accepted non-AutoAR binary")
	}
	if err := validateAIAgentCommand("run_shell", []string{"curl", strings.Repeat("x", 4097)}); err == nil {
		t.Fatal("oversized argument allowed")
	}
	_ = os.Getenv("AUTOAR_AI_EXECUTE_COMMANDS")
}
