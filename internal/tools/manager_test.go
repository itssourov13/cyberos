package tools

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/sandbox"
)

func TestStubRegisterExecuteViaBroker(t *testing.T) {
	mgr := NewManager(nil)
	if err := mgr.Register(NewStubScanner(nil)); err != nil {
		t.Fatal(err)
	}
	reg := execution.NewRegistry()
	if err := mgr.RegisterAllExecutors(reg); err != nil {
		t.Fatal(err)
	}
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	now := time.Now().UTC()
	grant := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "assessment.stub.probe",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		Constraints: contracts.ExecutionGrantConstraints{AllowedExecutorHints: []string{"stub.security.scanner"}},
	}
	res, err := brk.Execute(contracts.ExecutionRequest{
		Grant: grant, Parameters: map[string]any{"target": "example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != contracts.ExecutionStatusSucceeded {
		t.Fatalf("status=%s msg=%s", res.Status, res.ErrorMessage)
	}
	if res.Output["normalized"] != true {
		t.Fatalf("output=%v", res.Output)
	}
}

func TestUnknownToolFailsClosed(t *testing.T) {
	mgr := NewManager(nil)
	_, err := mgr.Get("nmap")
	if err == nil {
		t.Fatal("unknown tool must fail closed")
	}
}

func TestHealth(t *testing.T) {
	mgr := NewManager(nil)
	_ = mgr.Register(NewStubScanner(nil))
	h := mgr.HealthAll(context.Background())
	if h["stub.security.scanner"].Status != HealthHealthy {
		t.Fatalf("%v", h)
	}
}

func TestCapabilityMismatchRejected(t *testing.T) {
	mgr := NewManager(nil)
	_ = mgr.Register(NewStubScanner(nil))
	ex, _ := mgr.ExecutorFor("stub.security.scanner")
	now := time.Now().UTC()
	_, err := ex.Execute(contracts.ExecutionRequest{Grant: contracts.ExecutionGrant{
		ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "not.declared",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}})
	if err == nil {
		t.Fatal("capability mismatch must fail")
	}
}
