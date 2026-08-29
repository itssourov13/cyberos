package execution_test

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/resultauth"
	"cyberos/internal/storage/memory"
)

// E2E unit path: Grant-bound assignment → claim → signed result → verify → complete.
// Does not bypass Broker (Grant assumed issued); verifies binding chain integrity.
func TestPhase22D_AssignmentResultChain(t *testing.T) {
	repo := memory.NewAssignmentRepo()
	svc := execution.NewAssignmentService(repo)
	ctx := context.Background()

	dig := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	g := contracts.ExecutionGrant{
		ID: "grant-e2e", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-a", ScopeID: "scope-1", TaskID: "task-1", CapabilityID: "cap.demo",
		Status: contracts.ExecutionGrantStatusIssued, IssuedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour), Nonce: "n1",
		PackContentDigest: dig,
	}
	a, err := svc.CreateFromGrant(ctx, g, "h1", "w1", "asg-e2e")
	if err != nil {
		t.Fatal(err)
	}
	if a.GrantID != "grant-e2e" || a.PackContentDigest != dig {
		t.Fatalf("%+v", a)
	}
	a, err = svc.Dispatch(ctx, "proj-a", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, err = svc.Claim(ctx, "proj-a", a.ID, "w1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	pub, priv, _ := ed25519.GenerateKey(nil)
	keys := resultauth.NewKeyRegistry()
	_ = keys.Register("w1", pub)
	body := "ok"
	rh := resultauth.HashPayload("SUCCEEDED", body)
	env, err := resultauth.Sign(resultauth.Envelope{
		AssignmentID: a.ID, GrantID: a.GrantID, ProjectID: string(a.ProjectID),
		WorkerID: a.WorkerID, HostID: string(a.HostID), PackContentDigest: a.PackContentDigest,
		ResultHash: rh, ResultStatus: "SUCCEEDED", ResultID: "rid-1",
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
	}, priv)
	if err != nil {
		t.Fatal(err)
	}
	bind := resultauth.AssignmentBinding{
		AssignmentID: a.ID, GrantID: a.GrantID, ProjectID: string(a.ProjectID),
		WorkerID: a.WorkerID, HostID: string(a.HostID), PackContentDigest: a.PackContentDigest,
	}
	if err := resultauth.VerifyCompletion(env, bind, keys); err != nil {
		t.Fatal(err)
	}
	rg := resultauth.NewReplayGuard()
	if err := rg.Accept(env.ResultID); err != nil {
		t.Fatal(err)
	}
	if err := rg.Accept(env.ResultID); err == nil {
		t.Fatal("replay")
	}
	// pack mismatch in result
	bad := env
	bad.PackContentDigest = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	bad, _ = resultauth.Sign(bad, priv)
	if err := resultauth.VerifyCompletion(bad, bind, keys); err == nil {
		t.Fatal("pack A→B must deny")
	}
	a, err = svc.Complete(ctx, "proj-a", a.ID, "w1", rh, "SUCCEEDED", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != execution.AssignCompleted {
		t.Fatal(a.Status)
	}
}
