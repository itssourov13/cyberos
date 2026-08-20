package security_test

import (
	"context"
	"testing"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/tools"
	"cyberos/internal/tools/nmap"
	"cyberos/internal/tools/packs/recon"
)

func TestPlatform_RegisterDiscoverPack(t *testing.T) {
	p := tools.NewPlatform(nil)
	if err := recon.Register(p); err != nil {
		t.Fatal(err)
	}
	if len(p.Discover()) < 1 {
		t.Fatal("discover")
	}
	st := p.HealthSweep(context.Background())
	if _, ok := st[nmap.ToolID]; !ok {
		t.Fatal("nmap status")
	}
	pack, err := p.Packs().Get(tools.PackRecon)
	if err != nil || len(pack.ToolIDs) < 1 {
		t.Fatalf("%v %+v", err, pack)
	}
}

func TestPlatform_NoGrantNoExecution(t *testing.T) {
	p := tools.NewPlatform(nil)
	_ = recon.Register(p)
	reg := execution.NewRegistry()
	_ = p.WireExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("no grant")
	}
}

func TestPlatform_UnknownFailClosed(t *testing.T) {
	p := tools.NewPlatform(nil)
	if _, err := p.Resolve("tool.unknown"); err == nil {
		t.Fatal()
	}
}

func TestManifestRuntimes(t *testing.T) {
	for _, rt := range tools.RuntimeLabels() {
		m := tools.Manifest{ID: "x", Name: "X", Version: "1", Runtime: rt, ExecutorID: "x"}
		if err := m.Validate(); err != nil {
			t.Fatal(rt, err)
		}
	}
}
