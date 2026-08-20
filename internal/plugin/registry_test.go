package plugin

import (
	"context"
	"testing"

	"cyberos/internal/capability"
	"cyberos/internal/contracts"
)

func TestRegisterActivatePrepare(t *testing.T) {
	capReg := capability.NewRegistry(nil)
	reg := NewRegistry(capReg)
	a, err := NewBaseAdapter(contracts.PluginManifest{
		ID: "demo-adapter", Name: "Demo", Version: "1",
		Runtime:      contracts.PluginRuntimeGo,
		Capabilities: []contracts.CapabilityID{"tool.demo.ping"},
		ExecutorHint: "mock",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if err := reg.Activate(context.Background(), "demo-adapter"); err != nil {
		t.Fatal(err)
	}
	got, err := reg.Get("demo-adapter")
	if err != nil || got.State != contracts.PluginStateActive {
		t.Fatalf("state=%v err=%v", got.State, err)
	}
	params, err := reg.Prepare(context.Background(), "demo-adapter", "tool.demo.ping", map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	if params["plugin_id"] != "demo-adapter" {
		t.Fatalf("%v", params)
	}
	// Capability discovery registered
	if _, err := capReg.RequireKnown(context.Background(), "tool.demo.ping"); err != nil {
		t.Fatalf("capability not discovered: %v", err)
	}
}

func TestDuplicateRegister(t *testing.T) {
	reg := NewRegistry(nil)
	a, _ := NewBaseAdapter(contracts.PluginManifest{
		ID: "x", Name: "X", Version: "1", Runtime: contracts.PluginRuntimePython,
	})
	_ = reg.Register(context.Background(), a)
	if err := reg.Register(context.Background(), a); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestPrepareRequiresActive(t *testing.T) {
	reg := NewRegistry(nil)
	a, _ := NewBaseAdapter(contracts.PluginManifest{
		ID: "y", Name: "Y", Version: "1", Runtime: contracts.PluginRuntimeRust,
		Capabilities: []contracts.CapabilityID{"c1"},
	})
	_ = reg.Register(context.Background(), a)
	_, err := reg.Prepare(context.Background(), "y", "c1", nil)
	if err == nil {
		t.Fatal("inactive prepare must fail")
	}
}

func TestDisable(t *testing.T) {
	reg := NewRegistry(nil)
	a, _ := NewBaseAdapter(contracts.PluginManifest{
		ID: "z", Name: "Z", Version: "1", Runtime: contracts.PluginRuntimeGo,
	})
	_ = reg.Register(context.Background(), a)
	_ = reg.Activate(context.Background(), "z")
	_ = reg.Disable("z")
	got, _ := reg.Get("z")
	if got.State != contracts.PluginStateDisabled {
		t.Fatalf("state=%s", got.State)
	}
}
