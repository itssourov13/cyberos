package execution_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
)

type countingExec struct {
	n atomic.Int32
}

func (c *countingExec) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	c.n.Add(1)
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusSucceeded, StartedAt: now, FinishedAt: now, ExecutorID: "count",
	}, nil
}

func validGrant() contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID: "g1", Status: contracts.ExecutionGrantStatusIssued,
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-a", ScopeID: "s1",
		TaskID: "t1", CapabilityID: "cap.demo", Nonce: "n-" + now.Format("150405.000"),
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
}

func TestLocalHostImplementsExecutionHost(t *testing.T) {
	var _ execution.ExecutionHost = execution.NewLocalHost(&countingExec{})
}

func TestHostRegistry_DefaultLocal(t *testing.T) {
	reg := execution.NewHostRegistry()
	ex := &countingExec{}
	if err := reg.Register(execution.NewLocalHost(ex)); err != nil {
		t.Fatal(err)
	}
	h, err := reg.Resolve(contracts.ExecutionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if h.ID() != execution.HostLocal {
		t.Fatal(h.ID())
	}
}

func TestHostRegistry_UnknownHostDenied(t *testing.T) {
	reg := execution.NewHostRegistry()
	_ = reg.Register(execution.NewLocalHost(&countingExec{}))
	_, err := reg.Resolve(contracts.ExecutionRequest{HostID: "remote-attacker"})
	if err == nil {
		t.Fatal("expected deny")
	}
}

func TestHostRegistry_URLHostIDRejected(t *testing.T) {
	for _, bad := range []string{"https://attacker", "http://x", "ssh://root@h", "docker://x"} {
		if err := execution.ValidateHostID(execution.HostID(bad)); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestBroker_InvalidGrant_NoHostExecute(t *testing.T) {
	ex := &countingExec{}
	b := broker.New(ex)
	_, err := b.Execute(contracts.ExecutionRequest{Grant: contracts.ExecutionGrant{}})
	if err == nil {
		t.Fatal("expected invalid grant")
	}
	if ex.n.Load() != 0 {
		t.Fatalf("host must not execute on invalid grant; calls=%d", ex.n.Load())
	}
}

func TestBroker_UnknownHost_NoExecute(t *testing.T) {
	ex := &countingExec{}
	b := broker.New(ex)
	g := validGrant()
	_, err := b.Execute(contracts.ExecutionRequest{Grant: g, HostID: "https://attacker.example"})
	if err == nil {
		t.Fatal("expected unknown/invalid host")
	}
	if ex.n.Load() != 0 {
		t.Fatal("executor must not run")
	}
}

func TestBroker_ExplicitLocal(t *testing.T) {
	ex := &countingExec{}
	b := broker.New(ex)
	g := validGrant()
	res, err := b.Execute(contracts.ExecutionRequest{Grant: g, HostID: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != contracts.ExecutionStatusSucceeded {
		t.Fatal(res)
	}
	if res.HostID != "local" && res.HostID != "" {
		// HostID stamped by broker
		t.Log(res.HostID)
	}
	if ex.n.Load() != 1 {
		t.Fatal(ex.n.Load())
	}
}

func TestBroker_DefaultHostCompatible(t *testing.T) {
	ex := &countingExec{}
	b := broker.New(ex)
	g := validGrant()
	_, err := b.Execute(contracts.ExecutionRequest{Grant: g})
	if err != nil {
		t.Fatal(err)
	}
	if ex.n.Load() != 1 {
		t.Fatal("default local path broken")
	}
}

func TestLocalHost_NoAuth(t *testing.T) {
	// LocalHost executes without validating grant — authorization is Broker-only.
	ex := &countingExec{}
	h := execution.NewLocalHost(ex)
	_, _ = h.Execute(context.Background(), contracts.ExecutionRequest{})
	if ex.n.Load() != 1 {
		t.Fatal("LocalHost is adapter only")
	}
}
