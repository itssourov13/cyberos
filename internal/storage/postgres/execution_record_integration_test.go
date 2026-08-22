//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/provenance"
	"cyberos/internal/storage/postgres"
)

func TestPhase9D_ExecutionRecordDurability(t *testing.T) {
	if os.Getenv("CYBEROS_DATABASE_URL") == "" {
		// setupDB may still skip
	}
	db := setupDB(t)
	repo := postgres.NewExecutionRecordRepo(db)
	ctrl := lifecycle.New(broker.New(sandbox.New(mock.New())),
		lifecycle.WithEvidence(evidence.NewService()),
		lifecycle.WithProvenance(provenance.NewService()),
		lifecycle.WithRepository(repo),
	)
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "g-9d", TaskID: "t-9d", ProjectID: "p-9d", ScopeID: "s-9d", CapabilityID: "cap-1",
		Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		Status:    contracts.ExecutionGrantStatusIssued, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
		Nonce: "n-9d", CorrelationID: "corr-9d",
	}
	out, err := ctrl.Run(lifecycle.RunRequest{Grant: g, IdempotencyKey: "idem-9d", CorrelationID: "corr-9d"})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	db2 := setupDB(t)
	repo2 := postgres.NewExecutionRecordRepo(db2)
	got, err := repo2.GetByID(context.Background(), out.Record.ID)
	if err != nil || got.TaskID != "t-9d" {
		t.Fatal(err, got)
	}
	got2, err := repo2.GetByIdempotencyKey(context.Background(), "idem-9d")
	if err != nil || got2.ID != out.Record.ID {
		t.Fatal(err, got2)
	}
}
