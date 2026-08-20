package security_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/execution/process"
)

func TestSecurity_NoArbitraryShell(t *testing.T) {
	r := process.NewRunner(process.Policy{
		Enabled: true, AllowedBins: []string{"sh", "bash", "/bin/sh"},
		AllowShell: false, DefaultTimeout: time.Second,
	})
	_, err := r.RunDetailed(context.Background(), process.Spec{
		Path: "/bin/sh", Args: []string{"-c", "id"},
	})
	if err == nil {
		t.Fatal("shell must be rejected")
	}
}

func TestSecurity_UnknownBinaryRejected(t *testing.T) {
	r := process.NewRunner(process.Policy{Enabled: true, AllowedBins: []string{"true"}})
	_, err := r.RunDetailed(context.Background(), process.Spec{Path: "/bin/false"})
	if err == nil {
		t.Fatal("not allowlisted")
	}
}

func TestSecurity_ArgvInjectionRejected(t *testing.T) {
	if err := process.ValidateArgv("/bin/true", []string{"x;rm -rf /"}); err == nil {
		t.Fatal("injection")
	}
}
