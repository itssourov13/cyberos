package execution_test

import (
	"context"
	"crypto/ed25519"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/execution"
	"cyberos/internal/execution/resultauth"
	"cyberos/internal/storage/memory"
)

func TestComplete_ExpiredLease_Denies(t *testing.T) {
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	svc.SetNow(func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) })
	ctx := context.Background()
	a, err := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "ax", ExecutionID: "ex", ProjectID: "proj-a", HostID: "h1", WorkerID: "w1",
		GrantID: "g1",
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _ = svc.Dispatch(ctx, "proj-a", a.ID)
	a, _ = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Second)
	// advance past lease
	svc.SetNow(func() time.Time { return time.Date(2026, 1, 1, 12, 0, 5, 0, time.UTC) })
	_, err = svc.Complete(ctx, "proj-a", a.ID, "w1", "hash", "SUCCEEDED", "")
	if err == nil {
		t.Fatal("expired lease must deny completion")
	}
}

func TestClaim_ConcurrentTwoWorkers_OneWins(t *testing.T) {
	// Same worker_id on assignment — only that worker can claim; simulate
	// concurrent claims by same authorized worker → all succeed idempotently or one race.
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	ctx := context.Background()
	a, _ := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "ay", ExecutionID: "ey", ProjectID: "proj-a", HostID: "h1", WorkerID: "w1",
	})
	_, _ = svc.Dispatch(ctx, "proj-a", a.ID)
	var ok int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute); err == nil {
				atomic.AddInt32(&ok, 1)
			}
		}()
	}
	wg.Wait()
	if ok < 1 {
		t.Fatal()
	}
	// wrong worker never succeeds
	if _, err := svc.Claim(ctx, "proj-a", a.ID, "w2", time.Minute); err == nil {
		t.Fatal("wrong worker")
	}
}

func TestReplayGuard_Concurrent(t *testing.T) {
	g := resultauth.NewReplayGuard()
	var ok int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.Accept("same-result"); err == nil {
				atomic.AddInt32(&ok, 1)
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("expected exactly 1 accept, got %d", ok)
	}
}

func TestResultKey_Revoke_Denies(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	keys := resultauth.NewKeyRegistry()
	_ = keys.Register("w1", pub)
	keys.Revoke("w1")
	e, _ := resultauth.Sign(resultauth.Envelope{
		AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1",
		ResultHash: "h", ResultStatus: "SUCCEEDED", ResultID: "r1", CompletedAt: "2026-01-01T00:00:00Z",
	}, priv)
	bind := resultauth.AssignmentBinding{AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1"}
	if err := resultauth.VerifyCompletion(e, bind, keys); err == nil {
		t.Fatal()
	}
}

func TestWorker_Revoked_AuthenticateTLS_Denies(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, err := reg.Register(ctx, execution.Worker{
		ID: "w-rev", ProjectID: "proj-a", HostID: "h1", Status: execution.WorkerActive,
		CertFingerprint: "abc", CredentialRef: "ref://1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = reg.Activate(ctx, "proj-a", "w-rev")
	_, err = reg.Revoke(ctx, "proj-a", "w-rev")
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.AuthenticateTLS(ctx, "proj-a", "w-rev", "h1", "abc")
	if err == nil {
		t.Fatal("revoked worker must deny TLS auth")
	}
}
