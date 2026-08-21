package process

import (
	"testing"

	"cyberos/internal/config"
)

func TestPolicyFromConfig_FailClosedDefault(t *testing.T) {
	p := PolicyFromConfig(nil)
	if p.Enabled || p.AllowShell {
		t.Fatal(p)
	}
	p = PolicyFromConfig(&config.Config{})
	if p.Enabled || p.AllowShell {
		t.Fatal(p)
	}
}

func TestPolicyFromConfig_ExplicitEnable(t *testing.T) {
	p := PolicyFromConfig(&config.Config{
		ProcessEnabled: true, ProcessAllowedBins: []string{"/bin/true"},
		ProcessTimeout: "2s", ProcessMaxOutput: 4096,
	})
	if !p.Enabled || len(p.AllowedBins) != 1 || p.MaxOutputBytes != 4096 {
		t.Fatal(p)
	}
	// Enabled false forces shell off even if requested
	p = PolicyFromConfig(&config.Config{ProcessEnabled: false, ProcessAllowShell: true})
	if p.AllowShell {
		t.Fatal("shell must stay off when process disabled")
	}
}
