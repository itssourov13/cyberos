package execution_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/storage/memory"
)

func TestAssignmentLifecycle(t *testing.T) {
	svc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	ctx := context.Background()
	a, err := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "asg-1", ExecutionID: "ex-1", ProjectID: "proj-a", WorkerID: "w1", HostID: "worker-host-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err = svc.Dispatch(ctx, "proj-a", a.ID)
	if err != nil || a.Status != execution.AssignDispatched {
		t.Fatal(err, a.Status)
	}
	a, err = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
	if err != nil || a.Status != execution.AssignClaimed {
		t.Fatal(err, a.Status)
	}
	// wrong worker
	_, err = svc.Claim(ctx, "proj-a", a.ID, "w2", time.Minute)
	if err == nil {
		t.Fatal("wrong worker")
	}
	hash := execution.HashResult("SUCCEEDED", "out")
	a, err = svc.Complete(ctx, "proj-a", a.ID, "w1", hash, "SUCCEEDED", "")
	if err != nil || a.Status != execution.AssignCompleted {
		t.Fatal(err, a.Status)
	}
	// idempotent
	_, err = svc.Complete(ctx, "proj-a", a.ID, "w1", hash, "SUCCEEDED", "")
	if err != nil {
		t.Fatal(err)
	}
	// conflict
	_, err = svc.Complete(ctx, "proj-a", a.ID, "w1", "otherhash", "SUCCEEDED", "")
	if err == nil {
		t.Fatal("conflict")
	}
}

func TestAssignment_CancelWins(t *testing.T) {
	svc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	ctx := context.Background()
	a, _ := svc.Create(ctx, execution.ExecutionAssignment{
		ID: "asg-2", ExecutionID: "ex-2", ProjectID: "proj-a", WorkerID: "w1", HostID: "h1",
	})
	_, _ = svc.Dispatch(ctx, "proj-a", a.ID)
	a, err := svc.Cancel(ctx, "proj-a", a.ID)
	if err != nil || a.Status != execution.AssignCancelled {
		t.Fatal(err, a.Status)
	}
	_, err = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
	if err == nil {
		t.Fatal("cancelled cannot claim")
	}
}

func TestRemoteHost_GrantFirst(t *testing.T) {
	wrepo := memory.NewWorkerRepo()
	wreg := execution.NewWorkerRegistry(wrepo)
	ctx := context.Background()
	_, _ = wreg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "worker-host-01",
		CredentialRef: "secret://w1", Status: execution.WorkerActive,
	})
	_, _ = wreg.Activate(ctx, "proj-a", "w1")

	asvc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	rh, err := execution.NewRemoteHost("worker-host-01", wreg, asvc)
	if err != nil {
		t.Fatal(err)
	}
	rh.AutoComplete = true

	reg := execution.NewHostRegistry()
	_ = reg.Register(execution.NewLocalHost(&countingExec{}))
	_ = reg.Register(rh)
	b := broker.NewWithHosts(reg)

	// no grant
	_, err = b.Execute(contracts.ExecutionRequest{
		HostID: "worker-host-01",
		Parameters: map[string]any{"worker_id": "w1"},
	})
	if err == nil {
		t.Fatal("invalid grant must deny")
	}

	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "g-r1", Status: contracts.ExecutionGrantStatusIssued,
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-a", ScopeID: "s1", TaskID: "t1", CapabilityID: "cap.demo",
		Nonce: "nr-" + now.Format("150405.000000"), IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	res, err := b.Execute(contracts.ExecutionRequest{
		Grant: g, HostID: "worker-host-01",
		Parameters: map[string]any{"worker_id": "w1", "execution_id": "ex-r1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HostID != "worker-host-01" {
		t.Fatal(res.HostID)
	}
}

func TestRemoteHost_RevokedWorker(t *testing.T) {
	wreg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = wreg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "worker-host-01", CredentialRef: "s://1", Status: execution.WorkerActive,
	})
	_, _ = wreg.Activate(ctx, "proj-a", "w1")
	_, _ = wreg.Revoke(ctx, "proj-a", "w1")
	asvc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	rh, _ := execution.NewRemoteHost("worker-host-01", wreg, asvc)
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "g1", Status: contracts.ExecutionGrantStatusIssued,
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-a", ScopeID: "s1", TaskID: "t1", CapabilityID: "c",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	_, err := rh.Execute(ctx, contracts.ExecutionRequest{Grant: g, Parameters: map[string]any{"worker_id": "w1"}})
	if err == nil {
		t.Fatal("revoked worker")
	}
}

func TestRemoteHost_IdempotentDispatch(t *testing.T) {
	wreg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = wreg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "worker-host-01", CredentialRef: "s://1", Status: execution.WorkerActive,
	})
	_, _ = wreg.Activate(ctx, "proj-a", "w1")
	asvc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	rh, _ := execution.NewRemoteHost("worker-host-01", wreg, asvc)
	rh.AutoComplete = true
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "g-idem", Status: contracts.ExecutionGrantStatusIssued,
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-a", ScopeID: "s1", TaskID: "t1", CapabilityID: "c",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	req := contracts.ExecutionRequest{Grant: g, Parameters: map[string]any{"worker_id": "w1", "execution_id": "ex-idem"}}
	_, err := rh.Execute(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rh.Execute(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
}
