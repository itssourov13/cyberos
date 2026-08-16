package findings

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
)

func TestCreateFindingWithEvidence(t *testing.T) {
	evSvc := evidence.NewService()
	grant := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	_, err := evSvc.RecordFromExecution(contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
		Output:     map[string]any{"finding_hint": "open_port"},
		ExecutorID: "mock",
	}, grant, "ev-1")
	if err != nil {
		t.Fatal(err)
	}

	fSvc := NewService(evSvc)
	f, err := fSvc.Create(CreateRequest{
		ID: "f-1", TaskID: "t1", ProjectID: "p1", ScopeID: "s1",
		Severity: contracts.FindingSeverityMedium, Title: "Open port",
		EvidenceIDs: []contracts.EvidenceID{"ev-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.EvidenceIDs) != 1 {
		t.Errorf("evidence refs = %v", f.EvidenceIDs)
	}
}

func TestCreateRejectsMissingEvidence(t *testing.T) {
	fSvc := NewService(evidence.NewService())
	_, err := fSvc.Create(CreateRequest{
		ID: "f-2", TaskID: "t1", ProjectID: "p1", ScopeID: "s1",
		Severity: contracts.FindingSeverityLow, Title: "X",
		EvidenceIDs: []contracts.EvidenceID{"missing"},
	})
	if err == nil {
		t.Fatal("missing evidence should fail")
	}
}

func TestCreateRejectsTaskMismatch(t *testing.T) {
	evSvc := evidence.NewService()
	grant := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	_, _ = evSvc.RecordFromExecution(contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), ExecutorID: "mock",
	}, grant, "ev-1")

	fSvc := NewService(evSvc)
	_, err := fSvc.Create(CreateRequest{
		ID: "f-3", TaskID: "other-task", ProjectID: "p1", ScopeID: "s1",
		Severity: contracts.FindingSeverityLow, Title: "X",
		EvidenceIDs: []contracts.EvidenceID{"ev-1"},
	})
	if err == nil {
		t.Fatal("task mismatch should fail")
	}
}
