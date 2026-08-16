package principal

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewHuman(t *testing.T) {
	p, err := NewHuman("user-1", "alice")
	if err != nil {
		t.Fatalf("NewHuman failed: %v", err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if p.Type != contracts.PrincipalTypeHuman {
		t.Error("expected HUMAN")
	}
	if !p.IsActive() {
		t.Error("should be active")
	}
}

func TestInvalidType(t *testing.T) {
	if _, err := New("id", contracts.PrincipalType("NOPE"), "name"); err == nil {
		t.Error("invalid type should fail")
	}
}
