package process

import (
	"context"
	"testing"
	"time"
)

func TestValidateArgvRejectsShellCompounds(t *testing.T) {
	if err := ValidateArgv("/usr/bin/tool", []string{"a&&b"}); err == nil {
		t.Fatal("compound")
	}
	if err := ValidateArgv("/usr/bin/tool", []string{"$(id)"}); err == nil {
		t.Fatal("subshell")
	}
	if err := ValidateArgv("tool;rm", nil); err == nil {
		t.Fatal("bad path")
	}
}

func TestShellBinaryBlocked(t *testing.T) {
	if !IsShellBinary("/bin/bash") {
		t.Fatal("bash")
	}
	r := NewRunner(Policy{Enabled: true, AllowedBins: []string{"bash", "/bin/bash"}, AllowShell: false})
	_, err := r.RunDetailed(context.Background(), Spec{Path: "/bin/bash", Args: []string{"-c", "echo hi"}})
	if err == nil {
		t.Fatal("shell must be blocked")
	}
}

func TestAllowlistedTrue(t *testing.T) {
	r := NewRunner(Policy{
		Enabled: true, AllowedBins: []string{"true", "/bin/true"},
		DefaultTimeout: 2 * time.Second,
	})
	res, err := r.RunDetailed(context.Background(), Spec{Path: "/bin/true"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.TimedOut {
		t.Fatalf("%+v", res)
	}
}

func TestSanitizeEnvDropsSecrets(t *testing.T) {
	env := SanitizeEnv(map[string]string{
		"PATH":      "/usr/bin",
		"API_TOKEN": "secret",
		"SAFE_FLAG": "1",
		"bad-key":   "x",
	})
	joined := ""
	for _, e := range env {
		joined += e + " "
	}
	if contains(joined, "secret") || contains(joined, "API_TOKEN") {
		t.Fatalf("secret leaked: %v", env)
	}
	if !contains(joined, "SAFE_FLAG=1") {
		t.Fatalf("safe missing: %v", env)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
