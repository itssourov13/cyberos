package execution_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/execution"
	"cyberos/internal/storage/memory"
)

func TestClaimAtomic_ConcurrentSameWorkerIdempotent(t *testing.T) {
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	ctx := context.Background()
	a, err := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "a1", ExecutionID: "e1", ProjectID: "proj-a", HostID: "h1", WorkerID: "w1",
		Status: execution.AssignPending, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err = svc.Dispatch(ctx, "proj-a", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var ok int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
			if err == nil {
				atomic.AddInt32(&ok, 1)
			}
		}()
	}
	wg.Wait()
	if ok < 1 {
		t.Fatal("at least one claim must succeed")
	}
	final, err := svc.Get(ctx, "proj-a", a.ID)
	if err != nil || final.Status != execution.AssignClaimed {
		t.Fatalf("%v %+v", err, final)
	}
}

func TestClaimAtomic_WrongWorkerDenied(t *testing.T) {
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	ctx := context.Background()
	a, err := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "a2", ExecutionID: "e2", ProjectID: "proj-a", HostID: "h1", WorkerID: "w1",
		Status: execution.AssignPending, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = svc.Dispatch(ctx, "proj-a", a.ID)
	_, err = svc.Claim(ctx, "proj-a", a.ID, "w2", time.Minute)
	if err == nil {
		t.Fatal("wrong worker must fail")
	}
}

func TestClaimAtomic_TerminalDenied(t *testing.T) {
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	ctx := context.Background()
	a, err := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "a3", ExecutionID: "e3", ProjectID: "proj-a", HostID: "h1", WorkerID: "w1",
		Status: execution.AssignPending, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _ = svc.Dispatch(ctx, "proj-a", a.ID)
	a, _ = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
	_, err = svc.Complete(ctx, "proj-a", a.ID, "w1", "hash1", "SUCCEEDED", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
	if err == nil {
		t.Fatal("terminal reclaim denied")
	}
}
