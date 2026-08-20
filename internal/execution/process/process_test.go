package process

import (
	"context"
	"testing"
)

func TestDefaultPolicyBlocks(t *testing.T) {
	r := NewRunner(DefaultPolicy())
	_, _, err := r.Run(context.Background(), Spec{Path: "/bin/true"})
	if err == nil {
		t.Fatal("expected forbidden")
	}
}

func TestAllowlistRequired(t *testing.T) {
	r := NewRunner(Policy{Enabled: true, AllowedBins: []string{"true"}, DefaultTimeout: 0})
	_, _, err := r.Run(context.Background(), Spec{Path: "/bin/false"})
	if err == nil {
		t.Fatal("not allowlisted")
	}
}
