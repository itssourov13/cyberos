package resource

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestValidateAndDefaults(t *testing.T) {
	p := contracts.DefaultResourcePolicy()
	if err := ValidatePolicy(p); err != nil {
		t.Fatal(err)
	}
	bad := contracts.ResourcePolicy{MaxDuration: -1}
	if err := ValidatePolicy(bad); err == nil {
		t.Fatal("negative duration")
	}
	huge := contracts.ResourcePolicy{MaxOutputBytes: 1 << 40}
	if err := ValidatePolicy(huge); err == nil {
		t.Fatal("output ceiling")
	}
	merged := MergeDefaults(contracts.ResourcePolicy{})
	if merged.MaxDuration == 0 || merged.AllowNetwork {
		t.Fatalf("defaults not applied: %+v", merged)
	}
}

func TestEstimateOutput(t *testing.T) {
	n := EstimateOutputBytes(map[string]any{"a": "hello"})
	if n < 5 {
		t.Errorf("estimate=%d", n)
	}
	_ = time.Second
}
