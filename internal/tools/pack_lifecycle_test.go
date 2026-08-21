package tools

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
)

type noopAdapter struct {
	d Descriptor
}

func (n noopAdapter) Descriptor() Descriptor { return n.d }
func (n noopAdapter) Health(context.Context) HealthReport {
	return HealthReport{Status: HealthHealthy, Message: "ok"}
}
func (n noopAdapter) Run(context.Context, contracts.CapabilityID, map[string]any) (NormalizedResult, error) {
	return NormalizedResult{Success: true}, nil
}

func TestPackValidateAndVersionConflict(t *testing.T) {
	r := NewPackRegistry()
	p := Pack{ID: PackRecon, Name: "Recon", Version: "1.0.0", State: PackStateActive, ToolIDs: []string{"tool.a"}}
	if err := r.Register(p); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(p); err != nil {
		t.Fatal("same version should be idempotent", err)
	}
	p2 := p
	p2.Version = "2.0.0"
	if err := r.Register(p2); !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := r.Register(Pack{ID: "", Name: "x", Version: "1"}); err == nil {
		t.Fatal("invalid")
	}
}

func TestPackListActiveAndSetState(t *testing.T) {
	r := NewPackRegistry()
	_ = r.Register(Pack{ID: "p1", Name: "P1", Version: "1.0.0", State: PackStateActive})
	_ = r.Register(Pack{ID: "p2", Name: "P2", Version: "1.0.0", State: PackStateDisabled})
	active := r.ListActive()
	if len(active) != 1 || active[0].ID != "p1" {
		t.Fatal(active)
	}
	if err := r.SetState("p2", PackStateActive); err != nil {
		t.Fatal(err)
	}
	if len(r.ListActive()) != 2 {
		t.Fatal(r.ListActive())
	}
}

func TestCapabilityMapPackToolCap(t *testing.T) {
	mgr := NewManager(nil)
	_ = mgr.Register(noopAdapter{d: Descriptor{
		ID: "tool.scan", Name: "scan", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.scan", Capabilities: []contracts.CapabilityID{"cap.scan.open"},
	}})
	plat := NewPlatform(mgr)
	_ = plat.RegisterPack(Pack{
		ID: PackRecon, Name: "Recon", Version: "1.2.0", State: PackStateActive,
		ToolIDs: []string{"tool.scan"}, CapabilityIDs: []contracts.CapabilityID{"cap.scan.open"},
	})
	m := plat.CapabilityMap()
	if len(m) != 1 || m[0].PackID != PackRecon || m[0].CapabilityID != "cap.scan.open" {
		t.Fatal(m)
	}
	if len(plat.LookupCapability("cap.scan.open")) != 1 {
		t.Fatal("lookup")
	}
	if len(plat.LookupCapability("cap.other")) != 0 {
		t.Fatal("unknown")
	}
}

func TestHealthSweepDiscoveryOnly(t *testing.T) {
	mgr := NewManager(nil)
	_ = mgr.Register(noopAdapter{d: Descriptor{
		ID: "tool.h", Name: "h", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.h", Capabilities: []contracts.CapabilityID{"cap.h"},
	}})
	plat := NewPlatform(mgr)
	st := plat.HealthSweep(context.Background())
	if st["tool.h"].Health.Status != HealthHealthy {
		t.Fatal(st)
	}
	// Discovery does not imply execution capability beyond listing.
	got, ok := plat.StatusOf("tool.h")
	if !ok || !got.Registered {
		t.Fatal(got, ok)
	}
}

func TestBackwardCompatiblePackWithoutVersion(t *testing.T) {
	r := NewPackRegistry()
	if err := r.Register(Pack{ID: PackNet, Name: "Network", ToolIDs: []string{"t1"}}); err != nil {
		t.Fatal(err)
	}
	p, _ := r.Get(PackNet)
	if p.Version != "1.0.0" || p.State != PackStateRegistered {
		t.Fatal(p)
	}
}
