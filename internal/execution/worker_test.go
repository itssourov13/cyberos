package execution_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/storage/memory"
)

func TestWorkerLifecycle(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	w, err := reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "worker-host-01",
		CredentialRef: "secret://workers/w1", CertFingerprint: "abc123",
		Status: execution.WorkerPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	w, err = reg.Activate(ctx, "proj-a", "w1")
	if err != nil || w.Status != execution.WorkerActive {
		t.Fatal(err, w.Status)
	}
	w, err = reg.Suspend(ctx, "proj-a", "w1")
	if err != nil || w.Status != execution.WorkerSuspended {
		t.Fatal(err, w.Status)
	}
	w, err = reg.Activate(ctx, "proj-a", "w1")
	if err != nil {
		t.Fatal(err)
	}
	w, err = reg.Revoke(ctx, "proj-a", "w1")
	if err != nil || w.Status != execution.WorkerRevoked {
		t.Fatal(err, w.Status)
	}
	_, err = reg.Activate(ctx, "proj-a", "w1")
	if err == nil {
		t.Fatal("revoked must not reactivate")
	}
}

func TestWorker_CrossProject(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", CredentialRef: "secret://x",
	})
	_, err := reg.Get(ctx, "proj-b", "w1")
	if err == nil {
		t.Fatal("cross-project must deny")
	}
}

func TestWorker_HeartbeatDoesNotAuthZ(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", CredentialRef: "secret://x", Status: execution.WorkerActive,
	})
	_, err := reg.Heartbeat(ctx, "proj-a", "w1")
	if err != nil {
		t.Fatal(err)
	}
	// Heartbeat is liveness only — RemoteHost still refuses execution
	rh, _ := execution.NewRemoteHost("worker-host-01")
	_, err = rh.Execute(ctx, contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("remote must not execute in 17C")
	}
}

func TestWorker_URLHostRejected(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	_, err := reg.Register(context.Background(), execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "https://attacker", CredentialRef: "secret://x",
	})
	if err == nil {
		t.Fatal("URL host id must deny")
	}
}

func TestWorker_AuthenticateFingerprint(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", CredentialRef: "secret://x",
		CertFingerprint: "fp-good", Status: execution.WorkerActive,
	})
	_, err := reg.Authenticate(ctx, "proj-a", "w1", "fp-bad")
	if err == nil {
		t.Fatal("mismatch must deny")
	}
	_, err = reg.Authenticate(ctx, "proj-a", "w1", "fp-good")
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidTransition_Terminal(t *testing.T) {
	if execution.ValidTransition(execution.WorkerRevoked, execution.WorkerActive) {
		t.Fatal()
	}
	if execution.ValidTransition(execution.WorkerExpired, execution.WorkerActive) {
		t.Fatal()
	}
}

func TestDuplicateHostID(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, err := reg.Register(ctx, execution.Worker{ID: "w1", ProjectID: "p", HostID: "h1", CredentialRef: "s://1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.Register(ctx, execution.Worker{ID: "w2", ProjectID: "p", HostID: "h1", CredentialRef: "s://2"})
	if err == nil {
		t.Fatal("duplicate host binding")
	}
}

func TestWorker_Expired(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, err := reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", CredentialRef: "secret://x",
		Status: execution.WorkerActive, ExpiresAt: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.Heartbeat(ctx, "proj-a", "w1")
	if err == nil {
		t.Fatal("expired must deny heartbeat")
	}
}
