package tools_test

import (
	"os"
	"path/filepath"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
	"cyberos/internal/tools/schema"
	"cyberos/internal/workflow"
)

func TestLoadValidPackFile(t *testing.T) {
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	pf := tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack: tools.Pack{
			ID: "pack.external", Name: "External", Version: "1.0.0", State: tools.PackStateActive,
		},
		Tools: []tools.PackTool{{
			ID: "tool.echo", Name: "Echo", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
			ExecutorID: "tool.echo", Binary: "echo",
			Capabilities: []contracts.CapabilityID{"cap.echo"},
			Schemas: &schema.ToolSchemas{
				Parameters: &schema.ParameterSchema{
					Version: schema.SchemaV1,
					Fields:  []schema.ParamField{{Name: "msg", Type: schema.ParamString, Required: true}},
				},
				Argv: &schema.ArgvMapping{Version: schema.SchemaV1, Tokens: []string{"{{msg}}"}},
			},
		}},
	}
	res, err := loader.InstallPackFile(pf)
	if err != nil {
		t.Fatal(err)
	}
	if res.PackID != "pack.external" {
		t.Fatal(res)
	}
	d, err := plat.Catalog().Get("tool.echo")
	if err != nil {
		t.Fatal(err)
	}
	if d.Binary != "echo" {
		t.Fatal(d)
	}
	// Capability discovery
	r := workflow.NewCapabilityResolver(plat)
	rt, err := r.ResolvePrimary("cap.echo")
	if err != nil || rt.ToolID != "tool.echo" {
		t.Fatal(err, rt)
	}
}

func TestLoadInvalidFormatRejected(t *testing.T) {
	loader := tools.NewLoader(tools.NewPlatform(nil), nil)
	_, err := loader.InstallPackFile(tools.PackFile{
		FormatVersion: "99",
		Pack:          tools.Pack{ID: "p", Name: "P", Version: "1"},
	})
	if err == nil {
		t.Fatal("bad format")
	}
}

func TestLoadInvalidSchemaRejected(t *testing.T) {
	loader := tools.NewLoader(tools.NewPlatform(nil), nil)
	_, err := loader.InstallPackFile(tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack:          tools.Pack{ID: "p", Name: "P", Version: "1.0.0"},
		Tools: []tools.PackTool{{
			ID: "t", Name: "T", Version: "1", Runtime: contracts.PluginRuntimeGo, ExecutorID: "t",
			Schemas: &schema.ToolSchemas{
				Parameters: &schema.ParameterSchema{Version: "99"},
			},
		}},
	})
	if err == nil {
		t.Fatal("bad schema")
	}
}

func TestDuplicateVersionIdempotent(t *testing.T) {
	loader := tools.NewLoader(tools.NewPlatform(nil), nil)
	pf := tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack:          tools.Pack{ID: "pack.x", Name: "X", Version: "1.0.0"},
		Tools: []tools.PackTool{{
			ID: "tool.x", Name: "X", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo, ExecutorID: "tool.x",
		}},
	}
	if _, err := loader.InstallPackFile(pf); err != nil {
		t.Fatal(err)
	}
	if _, err := loader.InstallPackFile(pf); err != nil {
		t.Fatal("same version idempotent", err)
	}
	pf.Tools[0].Version = "2.0.0"
	if _, err := loader.InstallPackFile(pf); err == nil {
		t.Fatal("version conflict")
	}
}

func TestDisabledPackExcludedFromResolver(t *testing.T) {
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	pf := tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack: tools.Pack{
			ID: "pack.d", Name: "D", Version: "1.0.0", State: tools.PackStateDisabled,
			ToolIDs: []string{"tool.d"},
		},
		Tools: []tools.PackTool{{
			ID: "tool.d", Name: "D", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo, ExecutorID: "tool.d",
			Capabilities: []contracts.CapabilityID{"cap.d"},
		}},
	}
	if _, err := loader.InstallPackFile(pf); err != nil {
		t.Fatal(err)
	}
	r := workflow.NewCapabilityResolver(plat)
	if _, err := r.Resolve("cap.d"); err == nil {
		t.Fatal("disabled pack tools must not resolve")
	}
}

func TestLoadFromJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.pack.json")
	content := `{
		"format_version": "1",
		"pack": {"id": "pack.file", "name": "File", "version": "1.0.0", "state": "ACTIVE"},
		"tools": [{
			"id": "tool.file", "name": "File", "version": "1.0.0", "runtime": "go",
			"executor_id": "tool.file", "capabilities": ["cap.file"]
		}]
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	res, err := loader.LoadJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if res.PackID != "pack.file" {
		t.Fatal(res)
	}
}

func TestLoadDoesNotExecuteOrAllowlist(t *testing.T) {
	// Structural: InstallPackFile only touches PackRegistry + Catalog.
	// Process policy is never referenced by loader.go.
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	_, err := loader.InstallPackFile(tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack:          tools.Pack{ID: "pack.safe", Name: "S", Version: "1.0.0"},
		Tools: []tools.PackTool{{
			ID: "tool.safe", Name: "S", Version: "1", Runtime: contracts.PluginRuntimeGo, ExecutorID: "tool.safe",
			Binary:       "/bin/sh", // metadata only — not allowlisted by load
			Capabilities: []contracts.CapabilityID{"cap.safe"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Catalog has binary path as metadata; process policy unchanged (default disabled).
	d, _ := plat.Catalog().Get("tool.safe")
	if d.Binary != "/bin/sh" {
		t.Fatal(d)
	}
}
