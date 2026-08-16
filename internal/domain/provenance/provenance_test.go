package provenance

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewForEvidence(t *testing.T) {
	p, err := NewForEvidence(
		"ev-1",
		contracts.PrincipalRef{ID: "user-1", Type: contracts.PrincipalTypeHuman},
		"proj-1", "scope-1", "task-1", "cap-1", "grant-1", "mock",
	)
	if err != nil {
		t.Fatalf("NewForEvidence failed: %v", err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if p.SubjectType != "evidence" || p.SubjectID != "ev-1" {
		t.Error("subject mismatch")
	}
	if p.ExecutorID != "mock" {
		t.Error("executor mismatch")
	}
}
