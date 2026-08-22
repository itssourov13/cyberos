package schema_test

import (
	"testing"

	"cyberos/internal/tools/schema"
)

func TestParameterRequiredAndType(t *testing.T) {
	s := schema.ParameterSchema{
		Version: schema.SchemaV1,
		Fields: []schema.ParamField{
			{Name: "target", Type: schema.ParamString, Required: true, MaxLen: intPtr(64)},
			{Name: "mode", Type: schema.ParamEnum, Enum: []string{"list", "connect"}, Default: "list"},
		},
	}
	_, err := s.Apply(map[string]any{})
	if err == nil {
		t.Fatal("missing required")
	}
	out, err := s.Apply(map[string]any{"target": "example.com"})
	if err != nil || out["mode"] != "list" {
		t.Fatal(err, out)
	}
	_, err = s.Apply(map[string]any{"target": "x", "mode": "evil"})
	if err == nil {
		t.Fatal("enum")
	}
	_, err = s.Apply(map[string]any{"target": "x", "binary": "/bin/sh"})
	if err == nil {
		t.Fatal("binary override")
	}
}

func TestArgvMaterialize(t *testing.T) {
	m := schema.ArgvMapping{
		Version: schema.SchemaV1,
		Tokens:  []string{"-sL", "--", "{{target}}"},
	}
	args, err := m.Materialize(map[string]any{"target": "example.com"})
	if err != nil || len(args) != 3 || args[2] != "example.com" {
		t.Fatal(err, args)
	}
	_, err = m.Materialize(map[string]any{})
	if err == nil {
		t.Fatal("unknown var")
	}
	_, err = m.Materialize(map[string]any{"target": "a&&b"})
	if err == nil {
		t.Fatal("shell compound")
	}
}

func TestArgvForbiddenFlags(t *testing.T) {
	m := schema.ArgvMapping{Version: schema.SchemaV1, Tokens: []string{"-c", "id"}}
	if err := m.Validate(); err == nil {
		t.Fatal("-c")
	}
}

func TestUnknownSchemaVersion(t *testing.T) {
	s := schema.ParameterSchema{Version: "99"}
	if err := s.Validate(); err == nil {
		t.Fatal("version")
	}
	o := schema.OutputSchema{Version: "99"}
	if err := o.Validate(); err == nil {
		t.Fatal("output version")
	}
}

func TestOutputSchemaDataOnly(t *testing.T) {
	o := schema.OutputSchema{Version: schema.SchemaV1, Fields: []schema.OutputField{{Name: "summary"}}}
	if err := o.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestToolSchemasBuildArgs(t *testing.T) {
	ts := schema.ToolSchemas{
		Parameters: &schema.ParameterSchema{
			Version: schema.SchemaV1,
			Fields:  []schema.ParamField{{Name: "target", Type: schema.ParamString, Required: true}},
		},
		Argv: &schema.ArgvMapping{
			Version: schema.SchemaV1,
			Tokens:  []string{"--", "{{target}}"},
		},
	}
	args, err := ts.BuildArgs(map[string]any{"target": "host.example"})
	if err != nil || args[1] != "host.example" {
		t.Fatal(err, args)
	}
}

func intPtr(n int) *int { return &n }
