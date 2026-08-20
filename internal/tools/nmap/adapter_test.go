package nmap

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/process"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/tools"
)

func TestDescriptorAndCapability(t *testing.T) {
	a := New(Options{})
	d := a.Descriptor()
	if d.ID != ToolID || d.ExecutorID != ExecutorID {
		t.Fatalf("%+v", d)
	}
	if len(d.Capabilities) != 1 || d.Capabilities[0] != CapabilityID {
		t.Fatal(d.Capabilities)
	}
}

func TestHealthMissingBinary(t *testing.T) {
	a := New(Options{Binary: "/nonexistent/nmap-binary"})
	h := a.Health(context.Background())
	if h.Status != tools.HealthUnhealthy {
		t.Fatalf("%+v", h)
	}
}

func TestHealthDegradedWhenProcessDisabled(t *testing.T) {
	// May skip if nmap not installed
	if _, err := resolveBinary("nmap"); err != nil {
		t.Skip("nmap not installed")
	}
	a := New(Options{Binary: "nmap", Proc: process.NewRunner(process.DefaultPolicy())})
	h := a.Health(context.Background())
	if h.Status != tools.HealthDegraded && h.Status != tools.HealthHealthy {
		t.Fatalf("%+v", h)
	}
}

func TestBuildArgvRejectsInjection(t *testing.T) {
	_, _, err := buildArgv(map[string]any{"target": "host;rm -rf /"})
	if err == nil {
		t.Fatal("injection")
	}
	_, _, err = buildArgv(map[string]any{"target": "example.com", "mode": "connect", "ports": "80;id"})
	if err == nil {
		t.Fatal("ports injection")
	}
}

func TestUnknownCapabilityRejected(t *testing.T) {
	a := New(Options{Proc: process.NewRunner(process.DefaultPolicy())})
	_, err := a.Run(context.Background(), "not.a.cap", map[string]any{"target": "127.0.0.1"})
	if err == nil {
		t.Fatal("capability")
	}
}

func TestRegisterAndBrokerNoGrant(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(New(Options{}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("no grant")
	}
}

func TestBrokerPathWithGrantWhenDisabled(t *testing.T) {
	// Process disabled → execution fails closed even with grant
	proc := process.NewRunner(process.DefaultPolicy())
	mgr := tools.NewManager(proc)
	_ = mgr.Register(New(Options{Proc: proc}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	now := time.Now().UTC()
	res, err := brk.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g1", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: CapabilityID,
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
			Constraints: contracts.ExecutionGrantConstraints{AllowedExecutorHints: []string{ExecutorID}},
		},
		Parameters: map[string]any{"target": "127.0.0.1", "mode": "list"},
	})
	// Expect failure because process policy disabled or binary missing
	if err == nil && res.Status == contracts.ExecutionStatusSucceeded {
		t.Fatal("must not succeed with process disabled")
	}
}

func TestAllowlistedVersionProbe(t *testing.T) {
	path, err := resolveBinary("nmap")
	if err != nil {
		t.Skip("nmap not installed")
	}
	proc := process.NewRunner(process.Policy{
		Enabled: true, AllowedBins: []string{"nmap", path},
		DefaultTimeout: 5 * time.Second,
	})
	a := New(Options{Binary: path, Proc: proc})
	h := a.Health(context.Background())
	if h.Status != tools.HealthHealthy && h.Status != tools.HealthDegraded {
		t.Fatalf("%+v", h)
	}
}
