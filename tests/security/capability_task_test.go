package security_test

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
)

// Security invariant: capability discovery must never imply authorization.
// The registry exposes no Authorize method and RequireKnown only checks existence.
func TestSecurity_CapabilityDiscoveryNotAuthorization(t *testing.T) {
	reg := capability.NewRegistry(nil)
	c, _ := domaincap.New("dangerous.exploit", "Exploit", contracts.CapabilityTypeCustom)
	if err := reg.Register(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	// Known ≠ authorized
	got, err := reg.RequireKnown(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID {
		t.Error("id mismatch")
	}
	// There is intentionally no reg.Authorize(...) API.
}

func TestSecurity_UnknownCapabilityRejected(t *testing.T) {
	reg := capability.NewRegistry(nil)
	_, err := reg.RequireKnown(context.Background(), "not.registered")
	if !errors.Is(err, contracts.ErrCapabilityUnknown) {
		t.Fatalf("expected ErrCapabilityUnknown, got %v", err)
	}
}
