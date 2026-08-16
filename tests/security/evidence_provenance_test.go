package security_test

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/provenance"
)

// Security/integrity: evidence without full execution context must not be recordable.
func TestSecurity_EvidenceRequiresFullContext(t *testing.T) {
	svc := evidence.NewService()
	grant := contracts.ExecutionGrant{
		ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p", ScopeID: "s", TaskID: "", CapabilityID: "c",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	_, err := svc.RecordFromExecution(contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
	}, grant, "ev")
	if err == nil {
		t.Fatal("evidence without task linkage must fail")
	}
}

// Integrity: provenance must answer WHO/WHAT/WHEN/WHERE.
func TestSecurity_ProvenanceCompleteness(t *testing.T) {
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
	if err := evidence.EnsureLinked(ev); err != nil {
		t.Fatal(err)
	}

	pSvc := provenance.NewService()
	p, err := pSvc.RecordForEvidence(ev, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if p.Principal.ID == "" || p.ScopeID == "" || p.TaskID == "" ||
		p.CapabilityID == "" || p.ProducedAt.IsZero() {
		t.Fatalf("provenance incomplete: %+v", p)
	}
}
