package brain

import "testing"

func TestAIAgentCommandPolicy(t *testing.T) {
	t.Setenv("AUTOAR_AI_EXECUTE_COMMANDS", "")
	t.Setenv("AUTOAR_AI_ALLOWED_DOMAINS", "example.org")
	if err := validateAIAgentCommand("run_command", []string{"autoar", "domain", "run", "-d", "example.org"}); err == nil {
		t.Fatal("execution enabled by default")
	}

	t.Setenv("AUTOAR_AI_EXECUTE_COMMANDS", "true")
	for _, tc := range []struct{action string; args []string; allowed bool}{
		{"run_command", []string{"autoar", "status", "--json"}, true},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org"}, true},
		{"run_command", []string{"autoar", "subdomains", "get", "-d", "a.example.org"}, true},
		{"run_command", []string{"autoar", "ports", "scan", "-d", "example.org", "-t", "16"}, true},
		{"run_shell", []string{"curl", "https://example.org"}, false},
		{"run_shell", []string{"bash", "-c", "id"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "badexample.org"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org.evil"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "127.0.0.1"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "localhost"}, false},
		{"run_command", []string{"autoar", "explain", "/etc/shadow"}, false},
		{"run_command", []string{"autoar", "github", "scan", "-r", "attacker/repo"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org", "--output", "/tmp/a"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org", "-t", "500"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org;curl"}, false},
		{"run_command", []string{"autoar", "domain", "run", "-d", "example.org", "-s", "other.example.org"}, false},
	} {
		err := validateAIAgentCommand(tc.action, tc.args)
		if (err == nil) != tc.allowed { t.Errorf("%q %q: allowed=%t, err=%v", tc.action, tc.args, tc.allowed, err) }
	}
	t.Setenv("AUTOAR_AI_ALLOWED_DOMAINS", "")
	if err := validateAIAgentCommand("run_command", []string{"autoar", "domain", "run", "-d", "example.org"}); err == nil {
		t.Fatal("scanning must fail closed when scope is empty")
	}
}
