package capability

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewAndValidate(t *testing.T) {
	c, err := New("recon.domain.enumerate", "Domain Enumeration", contracts.CapabilityTypeRecon)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !c.IsActive() {
		t.Error("should be active")
	}
	d := c.Descriptor()
	if d.ID != c.ID || d.Name != c.Name {
		t.Error("descriptor mismatch")
	}
}

func TestInvalidType(t *testing.T) {
	if _, err := New("id", "name", contracts.CapabilityType("BAD")); err == nil {
		t.Error("bad type should fail")
	}
}
