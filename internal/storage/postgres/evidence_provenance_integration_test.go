//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
	"cyberos/internal/evidence"
	"cyberos/internal/provenance"
	"cyberos/internal/storage/postgres"
)

func TestPhase8A_EvidenceProvenanceDurability(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	eRepo := postgres.NewEvidenceRepo(db)
	pRepo := postgres.NewProvenanceRepo(db)

	evSvc := evidence.NewService().WithRepository(eRepo)
	prSvc := provenance.NewService().WithRepository(pRepo)

	now := time.Now().UTC()
	grant := contracts.ExecutionGrant{
		ID: "g-8a", TaskID: "t-8a", ProjectID: "p-8a", ScopeID: "s-8a",
		CapabilityID: "cap.8a",
		Principal:    contracts.PrincipalRef{ID: "u-8a", Type: contracts.PrincipalTypeHuman},
		Status:       contracts.ExecutionGrantStatusIssued, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	ev, err := evSvc.RecordFromExecution(contracts.ExecutionResult{
		Status: contracts.ExecutionStatusSucceeded,
		Output: map[string]any{"port": 22}, CorrelationID: "corr-8a", ExecutorID: "mock",
	}, grant, "ev-8a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prSvc.RecordForEvidence(ev, "mock"); err != nil {
		t.Fatal(err)
	}

	_ = db.Close()
	db2 := setupDB(t)
	e2 := evidence.NewService().WithRepository(postgres.NewEvidenceRepo(db2))
	p2 := provenance.NewService().WithRepository(postgres.NewProvenanceRepo(db2))

	got, err := e2.Get(contracts.EvidenceID("ev-8a"))
	if err != nil || got.ProjectID != "p-8a" {
		t.Fatal(err, got)
	}
	pr, err := p2.Get("evidence", "ev-8a")
	if err != nil || pr.CorrelationID != "corr-8a" {
		t.Fatal(err, pr)
	}
	_ = domainevidence.Evidence{}
}
