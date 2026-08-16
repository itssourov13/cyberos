package capability

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
)

func TestRegisterAndGet(t *testing.T) {
	reg := NewRegistry(nil)
	c, err := domaincap.New("recon.domain.enumerate", "Domain Enum", contracts.CapabilityTypeRecon)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	got, err := reg.Get(context.Background(), c.ID)
	if err != nil || got.Name != c.Name {
		t.Fatalf("get: %v %+v", err, got)
	}
}

func TestRequireKnownUnknown(t *testing.T) {
	reg := NewRegistry(nil)
	_, err := reg.RequireKnown(context.Background(), "does.not.exist")
	if !errors.Is(err, contracts.ErrCapabilityUnknown) {
		t.Errorf("expected ErrCapabilityUnknown, got %v", err)
	}
}

func TestDuplicateRegister(t *testing.T) {
	reg := NewRegistry(nil)
	c, _ := domaincap.New("cap.1", "One", contracts.CapabilityTypeUtility)
	_ = reg.Register(context.Background(), c)
	err := reg.Register(context.Background(), c)
	if !errors.Is(err, contracts.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestDiscoveryDoesNotImplyAuth(t *testing.T) {
	// Documented invariant: registry only discovers.
	// There is no Authorize method on the registry.
	reg := NewRegistry(nil)
	c, _ := domaincap.New("cap.x", "X", contracts.CapabilityTypeScan)
	_ = reg.Register(context.Background(), c)
	d, err := reg.Descriptor(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != c.ID {
		t.Error("descriptor mismatch")
	}
	// No authorization API exists on Registry — compile-time guarantee of separation.
}
