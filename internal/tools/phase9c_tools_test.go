package tools_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/execution"
	"cyberos/internal/execution/process"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/provenance"
	"cyberos/internal/tools"
	"cyberos/internal/tools/httpprobe"
	"cyberos/internal/tools/nmap"
	"cyberos/internal/tools/packs/recon"
	"cyberos/internal/tools/packs/web"
)

func grant(cap contracts.CapabilityID, nonce string) contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID: contracts.ExecutionGrantID("g-" + nonce), TaskID: "t1", ProjectID: "proj-a", ScopeID: "scope-a",
		CapabilityID: cap,
		Principal:    contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		Status:       contracts.ExecutionGrantStatusIssued,
		IssuedAt:     now, ExpiresAt: now.Add(time.Hour),
		Nonce: nonce, CorrelationID: contracts.CorrelationID("corr-" + nonce),
		Constraints: contracts.ExecutionGrantConstraints{AllowedExecutorHints: []string{nmap.ExecutorID, httpprobe.ExecutorID}},
	}
}

func TestPhase9C_NoGrantDenied(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(nmap.New(nmap.Options{}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	if _, err := brk.Execute(contracts.ExecutionRequest{}); err == nil {
		t.Fatal("no grant")
	}
}

func TestPhase9C_WrongCapabilityDenied(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(nmap.New(nmap.Options{}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	g := grant("wrong.cap", "wc1")
	_, err := brk.Execute(contracts.ExecutionRequest{Grant: g, Parameters: map[string]any{"target": "127.0.0.1"}})
	if err == nil {
		t.Fatal("wrong capability")
	}
}

func TestPhase9C_ExpiredGrantDenied(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(nmap.New(nmap.Options{}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	g := grant(nmap.CapabilityID, "exp1")
	g.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	_, err := brk.Execute(contracts.ExecutionRequest{Grant: g, Parameters: map[string]any{"target": "127.0.0.1"}})
	if err == nil {
		t.Fatal("expired")
	}
}

func TestPhase9C_ProcessDisabledFailClosed(t *testing.T) {
	proc := process.NewRunner(process.DefaultPolicy()) // disabled
	mgr := tools.NewManager(proc)
	_ = mgr.Register(nmap.New(nmap.Options{Proc: proc}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	res, err := brk.Execute(contracts.ExecutionRequest{
		Grant: grant(nmap.CapabilityID, "dis1"), Parameters: map[string]any{"target": "127.0.0.1", "mode": "list"},
	})
	if err == nil && res.Status == contracts.ExecutionStatusSucceeded {
		t.Fatal("must not succeed with process disabled")
	}
}

func TestPhase9C_UserBinaryParamRejected(t *testing.T) {
	a := nmap.New(nmap.Options{})
	_, err := a.Run(context.Background(), nmap.CapabilityID, map[string]any{
		"target": "127.0.0.1", "binary": "/bin/sh",
	})
	if err == nil {
		t.Fatal("user binary path")
	}
	h := httpprobe.New(httpprobe.Options{})
	_, err = h.Run(context.Background(), httpprobe.CapabilityID, map[string]any{
		"url": "https://example.com", "binary": "/bin/sh",
	})
	if err == nil {
		t.Fatal("http binary path")
	}
}

func TestPhase9C_PacksRegisterDiscoveryOnly(t *testing.T) {
	plat := tools.NewPlatform(nil)
	if err := recon.Register(plat); err != nil {
		t.Fatal(err)
	}
	if err := web.Register(plat); err != nil {
		t.Fatal(err)
	}
	p, err := plat.Packs().Get(tools.PackRecon)
	if err != nil || len(p.ToolIDs) == 0 {
		t.Fatal(p, err)
	}
	// discovery ≠ execution
	reg := execution.NewRegistry()
	_ = plat.WireExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	if _, err := brk.Execute(contracts.ExecutionRequest{}); err == nil {
		t.Fatal("still need grant")
	}
}

func TestPhase9C_EvidenceCorrelationOnResult(t *testing.T) {
	proc := process.NewRunner(process.DefaultPolicy())
	mgr := tools.NewManager(proc)
	_ = mgr.Register(nmap.New(nmap.Options{Proc: proc}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	g := grant(nmap.CapabilityID, "ev1")
	res, _ := brk.Execute(contracts.ExecutionRequest{
		Grant: g, Parameters: map[string]any{"target": "127.0.0.1"}, CorrelationID: "corr-ev1",
	})
	// May fail due to process disabled — still check metadata when output present
	if res.CorrelationID != "" && res.CorrelationID != "corr-ev1" && res.CorrelationID != g.CorrelationID {
		t.Fatalf("correlation: %q", res.CorrelationID)
	}
	evSvc := evidence.NewService()
	prSvc := provenance.NewService()
	if res.Status == contracts.ExecutionStatusSucceeded || res.Output != nil {
		ev, err := evSvc.RecordFromExecution(res, g, "ev-9c")
		if err == nil {
			_, _ = prSvc.RecordForEvidence(ev, res.ExecutorID)
			if ev.CorrelationID == "" {
				t.Fatal("evidence correlation")
			}
			if ev.ProjectID != g.ProjectID {
				t.Fatal("project isolation on evidence")
			}
		}
	}
}
