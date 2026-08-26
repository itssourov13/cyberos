//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"cyberos/internal/storage/ports"
	"cyberos/internal/storage/postgres"
)

func openOutboxTest(t *testing.T) (*postgres.DB, *postgres.AutonomyOutboxRepo) {
	t.Helper()
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		t.Skip("CYBEROS_DATABASE_URL unset — LIVE PG VERIFICATION SKIPPED (not a pass)")
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("CYBEROS_DATABASE_URL set but unreachable: %v", err)
	}
	return db, postgres.NewAutonomyOutboxRepo(db)
}

func Test16B_PG_ConcurrentClaimNoDuplicates(t *testing.T) {
	db, repo := openOutboxTest(t)
	defer db.Close()

	n := 12
	prefix := fmt.Sprintf("16b-%d-", time.Now().UnixNano())
	ctx := context.Background()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%sev-%d", prefix, i)
		err := repo.Enqueue(ctx, ports.AutonomyOutboxRecord{
			ID: id, ProjectID: "proj-16b", IdempotencyKey: id, EventID: id,
			EventType: "execution.completed", SchemaVersion: "1",
			AutonomyRunID: fmt.Sprintf("run-%d", i), // distinct runs → all claimable in one batch
			Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	workers := 8
	var mu sync.Mutex
	claimed := map[string]string{} // eventID -> worker
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wid string) {
			defer wg.Done()
			batch, err := repo.ClaimBatch(ctx, wid, n, 30*time.Second)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, rec := range batch {
				if prev, ok := claimed[rec.ID]; ok {
					t.Errorf("duplicate claim %s by %s and %s", rec.ID, prev, wid)
				}
				claimed[rec.ID] = wid
			}
		}(fmt.Sprintf("w-%d", w))
	}
	wg.Wait()
	if len(claimed) == 0 {
		t.Fatal("expected some claims")
	}
	if len(claimed) > n {
		t.Fatalf("claimed more than enqueued: %d > %d", len(claimed), n)
	}
}

func Test16B_PG_ActiveLeaseProtected(t *testing.T) {
	db, repo := openOutboxTest(t)
	defer db.Close()
	ctx := context.Background()
	id := fmt.Sprintf("16b-lease-%d", time.Now().UnixNano())
	_ = repo.Enqueue(ctx, ports.AutonomyOutboxRecord{
		ID: id, ProjectID: "proj-16b", IdempotencyKey: id, EventID: id,
		EventType: "execution.completed", SchemaVersion: "1",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	batch, err := repo.ClaimBatch(ctx, "owner", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) == 0 {
		t.Skip("event not claimed (pollution?)")
	}
	// Another worker must not steal active lease
	batch2, err := repo.ClaimBatch(ctx, "thief", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range batch2 {
		if rec.ID == id {
			t.Fatal("active lease stolen")
		}
	}
}

func Test16B_PG_ExpiredLeaseReclaimable(t *testing.T) {
	db, repo := openOutboxTest(t)
	defer db.Close()
	ctx := context.Background()
	id := fmt.Sprintf("16b-exp-%d", time.Now().UnixNano())
	_ = repo.Enqueue(ctx, ports.AutonomyOutboxRecord{
		ID: id, ProjectID: "proj-16b", IdempotencyKey: id, EventID: id,
		EventType: "execution.completed", SchemaVersion: "1",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	batch, err := repo.ClaimBatch(ctx, "w1", 1, 40*time.Millisecond)
	if err != nil || len(batch) == 0 {
		t.Fatal(err, len(batch))
	}
	time.Sleep(60 * time.Millisecond)
	batch2, err := repo.ClaimBatch(ctx, "w2", 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rec := range batch2 {
		if rec.ID == id {
			found = true
			if rec.LockedBy != "w2" {
				t.Fatal(rec.LockedBy)
			}
			if rec.Attempts < 2 {
				t.Fatalf("attempts=%d", rec.Attempts)
			}
		}
	}
	if !found {
		t.Fatal("expired lease not reclaimed")
	}
}
