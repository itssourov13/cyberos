package memory_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/storage/memory"
	"cyberos/internal/storage/ports"
)

func TestAutonomyRunPersistBudget(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	rec := ports.AutonomyRunRecord{
		ID: "r1", ProjectID: "proj-a", ScopeID: "s", State: "RUNNING",
		StepsConsumed: 7, Version: 1, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		PolicyJSON: []byte(`{"mode":"MANUAL"}`),
	}
	if err := repo.Create(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), "proj-a", "r1")
	if err != nil || got.StepsConsumed != 7 {
		t.Fatal(err, got)
	}
	// isolation
	_, err = repo.Get(context.Background(), "other", "r1")
	if err == nil {
		t.Fatal("cross project")
	}
	if err := repo.Cancel(context.Background(), "proj-a", "r1"); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.Get(context.Background(), "proj-a", "r1")
	if got.State != "CANCELLED" {
		t.Fatal(got.State)
	}
}

func TestVersionConflict(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	rec := ports.AutonomyRunRecord{ID: "r2", ProjectID: "p", State: "RUNNING", Version: 1, StartedAt: time.Now().UTC()}
	_ = repo.Create(context.Background(), rec)
	rec.Version = 99
	rec.StepsConsumed = 1
	if err := repo.Update(context.Background(), rec); err == nil {
		t.Fatal("expected conflict")
	}
}
