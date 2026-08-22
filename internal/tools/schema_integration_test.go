package tools_test

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
	"cyberos/internal/tools"
	"cyberos/internal/tools/schema"
)

func TestProcessAdapterUsesSchemas(t *testing.T) {
	d := tools.Descriptor{
		ID: "tool.demo", Name: "demo", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.demo", Binary: "/bin/true",
		Capabilities: []contracts.CapabilityID{"cap.demo"},
		Schemas: &schema.ToolSchemas{
			Parameters: &schema.ParameterSchema{
				Version: schema.SchemaV1,
				Fields:  []schema.ParamField{{Name: "target", Type: schema.ParamString, Required: true}},
			},
			Argv: &schema.ArgvMapping{
				Version: schema.SchemaV1,
				Tokens:  []string{"{{target}}"},
			},
		},
	}
	a, err := tools.NewProcessToolAdapter(d, process.NewRunner(process.DefaultPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	// process disabled → still validates schema before/during run
	_, err = a.Run(context.Background(), "cap.demo", map[string]any{})
	if err == nil {
		t.Fatal("missing param or disabled process")
	}
	_, err = a.Run(context.Background(), "cap.demo", map[string]any{"target": "ok", "binary": "/bin/sh"})
	if err == nil {
		t.Fatal("binary override")
	}
}

func TestSchemaRegistrationIsNotAuthorization(t *testing.T) {
	// Registering a schema-bearing descriptor does not enable process policy
	d := tools.Descriptor{
		ID: "tool.x", Name: "x", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.x", Binary: "/bin/true",
		Schemas: &schema.ToolSchemas{
			Argv: &schema.ArgvMapping{Version: schema.SchemaV1, Tokens: []string{"--version"}},
		},
	}
	mgr := tools.NewManager(nil)
	a, err := tools.NewProcessToolAdapter(d, process.NewRunner(process.DefaultPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	_ = mgr.Register(a)
	// Default policy disabled
	_, err = a.Run(context.Background(), "", map[string]any{})
	if err == nil {
		t.Fatal("must not execute without process enabled + grant path")
	}
}
