package config

import (
	"os"
	"testing"
)

func TestProcessPolicyDefaultsDisabled(t *testing.T) {
	t.Setenv("CYBEROS_PROCESS_ENABLED", "")
	t.Setenv("CYBEROS_PROCESS_ALLOW_SHELL", "")
	t.Setenv("CYBEROS_PROCESS_ALLOWED_BINS", "")
	c := Load()
	if c.ProcessEnabled {
		t.Fatal("process must default disabled")
	}
	if c.ProcessAllowShell {
		t.Fatal("shell must default disabled")
	}
}

func TestProcessPolicyFromEnv(t *testing.T) {
	t.Setenv("CYBEROS_PROCESS_ENABLED", "true")
	t.Setenv("CYBEROS_PROCESS_ALLOWED_BINS", "/bin/true,true")
	t.Setenv("CYBEROS_PROCESS_TIMEOUT", "5s")
	t.Setenv("CYBEROS_PROCESS_MAX_OUTPUT_BYTES", "1024")
	c := Load()
	if !c.ProcessEnabled {
		t.Fatal("enabled")
	}
	if len(c.ProcessAllowedBins) != 2 {
		t.Fatalf("bins: %v", c.ProcessAllowedBins)
	}
	if c.ProcessTimeout != "5s" || c.ProcessMaxOutput != 1024 {
		t.Fatalf("%q %d", c.ProcessTimeout, c.ProcessMaxOutput)
	}
	_ = os.Unsetenv
}
