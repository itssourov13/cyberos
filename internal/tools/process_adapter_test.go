package tools

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/process"
	"cyberos/internal/execution/sandbox"
)

func TestProcessAdapterAllowlistedTrue(t *testing.T) {
	proc := process.NewRunner(process.Policy{
		Enabled: true, AllowedBins: []string{"true", "/bin/true"},
		DefaultTimeout: 2 * time.Second,
	})
	a, err := NewProcessToolAdapter(Descriptor{
		ID: "demo.true", Name: "True", Version: "1",
		Runtime:      contracts.PluginRuntimeGo,
		Capabilities: []contracts.CapabilityID{"util.true"},
		ExecutorID:   "demo.true",
		Binary:       "/bin/true",
	}, proc)
	if err != nil {
		t.Fatal(err)
	}
	nr, err := a.Run(context.Background(), "util.true", map[string]any{})
	if err != nil || !nr.Success {
		t.Fatalf("run: %v %+v", err, nr)
	}

	mgr := NewManager(proc)
	_ = mgr.Register(a)
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	now := time.Now().UTC()
	res, err := brk.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "util.true",
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
			Constraints: contracts.ExecutionGrantConstraints{AllowedExecutorHints: []string{"demo.true"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != contracts.ExecutionStatusSucceeded {
		t.Fatalf("%s %s", res.Status, res.ErrorMessage)
	}
}

func TestProcessAdapterDisabledPolicy(t *testing.T) {
	a, err := NewProcessToolAdapter(Descriptor{
		ID: "demo.false", Name: "False", Version: "1",
		Runtime:      contracts.PluginRuntimeGo,
		Capabilities: []contracts.CapabilityID{"util.false"},
		ExecutorID:   "demo.false", Binary: "/bin/false",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Run(context.Background(), "util.false", nil)
	if err == nil {
		t.Fatal("disabled policy must fail")
	}
}
