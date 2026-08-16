package provenance

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/findings"
)

func TestRecordForEvidenceComplete(t *testing.T) {
	evSvc := evidence.NewService()
	grant := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	ev, err := evSvc.RecordFromExecution(contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), ExecutorID: "mock",
	}, grant, "ev-1")
	if err != nil {
		t.Fatal(err)
	}

	pSvc := NewService()
	p, err := pSvc.RecordForEvidence(ev, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if p.Principal.ID != "u1" || p.ScopeID != "s1" || p.TaskID != "t1" ||
		p.CapabilityID != "c1" || p.GrantID != "g1" {
		t.Errorf("incomplete provenance: %+v", p)
	}
	got, err := pSvc.Get("evidence", "ev-1")
	if err != nil || got.SubjectID != "ev-1" {
		t.Fatalf("get: %v", err)
	}
}

func TestRecordForFindingWithParents(t *testing.T) {
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

	fSvc := findings.NewService(evSvc)
	f, err := fSvc.Create(findings.CreateRequest{
		ID: "f-1", TaskID: "t1", ProjectID: "p1", ScopeID: "s1",
		Severity: contracts.FindingSeverityHigh, Title: "Issue",
		EvidenceIDs: []contracts.EvidenceID{"ev-1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	pSvc := NewService()
	p, err := pSvc.RecordForFinding(f,
		contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		"c1", "g1", "mock",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.ParentIDs) != 1 || p.ParentIDs[0] != "ev-1" {
		t.Errorf("parent ids = %v", p.ParentIDs)
	}
}
