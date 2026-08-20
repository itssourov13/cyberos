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
	"cyberos/internal/tools/httpprobe"
	"cyberos/internal/tools/nmap"
	"cyberos/internal/tools/packs/network"
	"cyberos/internal/tools/packs/recon"
	"cyberos/internal/tools/packs/vulnerability"
	"cyberos/internal/tools/packs/web"
	"cyberos/internal/tools/pythonprobe"
	"cyberos/internal/tools/rustprobe"
)

func TestPhase6C_MultiplePacks(t *testing.T) {
	p := tools.NewPlatform(nil)
	_ = recon.Register(p)
	_ = web.Register(p)
	_ = network.Register(p)
	_ = vulnerability.Register(p)
	packs := p.Packs().List()
	if len(packs) < 2 {
		t.Fatalf("packs=%d", len(packs))
	}
	ids := map[string]bool{}
	for _, pk := range packs {
		ids[pk.ID] = true
	}
	for _, need := range []string{tools.PackRecon, tools.PackWeb, tools.PackNet, tools.PackVuln} {
		if !ids[need] {
			t.Fatalf("missing pack %s", need)
		}
	}
}

func TestPhase6C_MultiRuntimeAdapters(t *testing.T) {
	p := tools.NewPlatform(nil)
	_ = network.Register(p)
	_ = web.Register(p)
	_ = recon.Register(p)
	runtimes := map[contracts.PluginRuntime]bool{}
	for _, d := range p.Discover() {
		runtimes[d.Runtime] = true
	}
	if !runtimes[contracts.PluginRuntimeGo] {
		t.Fatal("need go runtime (nmap or http)")
	}
	if !runtimes[contracts.PluginRuntimePython] {
		t.Fatal("need python probe")
	}
	if !runtimes[contracts.PluginRuntimeRust] {
		t.Fatal("need rust probe")
	}
}

func TestPhase6C_NoGrantAllAdapters(t *testing.T) {
	p := tools.NewPlatform(nil)
	_ = recon.Register(p)
	_ = web.Register(p)
	_ = network.Register(p)
	reg := execution.NewRegistry()
	_ = p.WireExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("no grant")
	}
}

func TestPhase6C_HTTPProbeInjection(t *testing.T) {
	a := httpprobe.New(httpprobe.Options{Proc: process.NewRunner(process.DefaultPolicy())})
	_, err := a.Run(context.Background(), httpprobe.CapabilityID, map[string]any{
		"url": "https://example.com;id",
	})
	if err == nil {
		t.Fatal("injection")
	}
}

func TestPhase6C_PythonCapabilityMismatch(t *testing.T) {
	a := pythonprobe.New(pythonprobe.Options{})
	_, err := a.Run(context.Background(), "wrong.cap", nil)
	if err == nil {
		t.Fatal()
	}
}

func TestPhase6C_NmapStillPresent(t *testing.T) {
	p := tools.NewPlatform(nil)
	_ = recon.Register(p)
	if _, err := p.Resolve(nmap.ToolID); err != nil {
		t.Fatal(err)
	}
	_ = time.Now()
	_ = rustprobe.ToolID
}
