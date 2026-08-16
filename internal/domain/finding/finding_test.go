package finding

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewAndAddEvidence(t *testing.T) {
	f, err := New("f-1", "task-1", "proj-1", "scope-1",
		contracts.FindingSeverityHigh, "Open redirect")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	f.AddEvidence("ev-1")
	f.AddEvidence("ev-2")
	if len(f.EvidenceIDs) != 2 {
		t.Errorf("expected 2 evidence ids, got %d", len(f.EvidenceIDs))
	}
}

func TestInvalidSeverity(t *testing.T) {
	_, err := New("f-1", "t", "p", "s", contracts.FindingSeverity("NOPE"), "title")
	if err == nil {
		t.Error("invalid severity should fail")
	}
}
