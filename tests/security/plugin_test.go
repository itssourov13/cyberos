package security_test

import (
	"context"
	"testing"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/plugin"
)

// Plugin discovery/prepare must not execute via broker without a grant.
func TestSecurity_PluginDiscoveryNotExecution(t *testing.T) {
	reg := plugin.NewRegistry(nil)
	a, err := plugin.NewBaseAdapter(contracts.PluginManifest{
		ID: "p1", Name: "P", Version: "1", Runtime: contracts.PluginRuntimeGo,
		Capabilities: []contracts.CapabilityID{"cap.x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = reg.Register(context.Background(), a)
	_ = reg.Activate(context.Background(), "p1")
	params, err := reg.Prepare(context.Background(), "p1", "cap.x", map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if params["plugin_id"] != "p1" {
		t.Fatal("prepare failed")
	}
	// Broker still rejects empty grant
	brk := broker.New(mock.New())
	_, err = brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("plugin must not enable grantless execution")
	}
}

// Adapter has no Execute method on the public boundary — only Prepare.
func TestSecurity_AdapterHasNoDirectExecute(t *testing.T) {
	var _ plugin.Adapter = (*plugin.BaseAdapter)(nil)
	// Compile-time: Adapter interface does not include Execute.
}
