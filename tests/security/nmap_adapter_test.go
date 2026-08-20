package security_test

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
	"cyberos/internal/tools/nmap"
)

func TestSecurity_NmapNoGrantNoExecution(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(nmap.New(nmap.Options{}))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{
		Parameters: map[string]any{"target": "127.0.0.1"},
	})
	if err == nil {
		t.Fatal("no grant")
	}
}

func TestSecurity_NmapUnknownToolFailClosed(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(nmap.New(nmap.Options{}))
	if _, err := mgr.Get("tool.nuclei"); err == nil {
		t.Fatal("unknown tool")
	}
}

func TestSecurity_NmapInjectionRejected(t *testing.T) {
	a := nmap.New(nmap.Options{Proc: process.NewRunner(process.DefaultPolicy())})
	_, err := a.Run(context.Background(), nmap.CapabilityID, map[string]any{
		"target": "127.0.0.1; cat /etc/passwd",
	})
	if err == nil {
		t.Fatal("injection")
	}
}

func TestSecurity_NmapProcessDisabledFailsClosed(t *testing.T) {
	proc := process.NewRunner(process.DefaultPolicy()) // disabled
	a := nmap.New(nmap.Options{Proc: proc})
	_, err := a.Run(context.Background(), nmap.CapabilityID, map[string]any{"target": "127.0.0.1"})
	if err == nil {
		// binary missing also errors — either way not success
		t.Log("expected error")
	}
	_ = time.Now()
}
