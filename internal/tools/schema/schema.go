// Package schema provides declarative parameter, argv, and output schemas for tools.
// Schemas validate and materialize data only — they never authorize or execute.
package schema

import (
	"fmt"
	"strconv"
	"strings"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
)

// SchemaVersion for parameter/argv/output schema documents.
const SchemaV1 contracts.SchemaVersion = "1"

// ParamType is a supported parameter primitive.
type ParamType string

const (
	ParamString  ParamType = "string"
	ParamInteger ParamType = "integer"
	ParamNumber  ParamType = "number"
	ParamBoolean ParamType = "boolean"
	ParamEnum    ParamType = "enum"
	ParamArray   ParamType = "array"
)

// ParamField describes one parameter.
type ParamField struct {
	Name     string    `json:"name"`
	Type     ParamType `json:"type"`
	Required bool      `json:"required,omitempty"`
	Default  any       `json:"default,omitempty"`
	Enum     []string  `json:"enum,omitempty"` // for ParamEnum
	MinLen   *int      `json:"min_len,omitempty"`
	MaxLen   *int      `json:"max_len,omitempty"`
	Min      *float64  `json:"min,omitempty"`
	Max      *float64  `json:"max,omitempty"`
}

// ParameterSchema is a versioned set of parameter fields.
type ParameterSchema struct {
	Version contracts.SchemaVersion `json:"version"`
	Fields  []ParamField            `json:"fields"`
}

// Validate checks schema structure.
func (s ParameterSchema) Validate() error {
	v := s.Version
	if v == "" {
		v = SchemaV1
	}
	if v != SchemaV1 {
		return fmt.Errorf("%w: unsupported parameter schema version %q", contracts.ErrInvalidArgument, s.Version)
	}
	seen := map[string]struct{}{}
	for _, f := range s.Fields {
		if f.Name == "" {
			return fmt.Errorf("%w: parameter field name required", contracts.ErrInvalidArgument)
		}
		if _, ok := seen[f.Name]; ok {
			return fmt.Errorf("%w: duplicate parameter %q", contracts.ErrInvalidArgument, f.Name)
		}
		seen[f.Name] = struct{}{}
		switch f.Type {
		case ParamString, ParamInteger, ParamNumber, ParamBoolean, ParamEnum, ParamArray:
		default:
			return fmt.Errorf("%w: unknown parameter type %q", contracts.ErrInvalidArgument, f.Type)
		}
		if f.Type == ParamEnum && len(f.Enum) == 0 {
			return fmt.Errorf("%w: enum parameter %q requires values", contracts.ErrInvalidArgument, f.Name)
		}
	}
	return nil
}

// Apply validates params against the schema and returns a normalized map (with defaults).
func (s ParameterSchema) Apply(params map[string]any) (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if params == nil {
		params = map[string]any{}
	}
	out := map[string]any{}
	for _, f := range s.Fields {
		val, ok := params[f.Name]
		if !ok || val == nil {
			if f.Required && f.Default == nil {
				return nil, fmt.Errorf("%w: missing required parameter %q", contracts.ErrInvalidArgument, f.Name)
			}
			if f.Default != nil {
				val = f.Default
				ok = true
			} else {
				continue
			}
		}
		n, err := coerceAndCheck(f, val)
		if err != nil {
			return nil, err
		}
		out[f.Name] = n
	}
	// Reject unknown keys that look like binary/path overrides
	for k := range params {
		if k == "binary" || k == "path" || k == "executable" {
			return nil, fmt.Errorf("%w: parameter %q cannot override executable path", contracts.ErrInvalidArgument, k)
		}
	}
	return out, nil
}

func coerceAndCheck(f ParamField, val any) (any, error) {
	switch f.Type {
	case ParamString, ParamEnum:
		s, ok := val.(string)
		if !ok {
			s = fmt.Sprint(val)
		}
		if f.Type == ParamEnum {
			found := false
			for _, e := range f.Enum {
				if e == s {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("%w: parameter %q value %q not in enum", contracts.ErrInvalidArgument, f.Name, s)
			}
		}
		if f.MinLen != nil && len(s) < *f.MinLen {
			return nil, fmt.Errorf("%w: parameter %q too short", contracts.ErrInvalidArgument, f.Name)
		}
		if f.MaxLen != nil && len(s) > *f.MaxLen {
			return nil, fmt.Errorf("%w: parameter %q too long", contracts.ErrInvalidArgument, f.Name)
		}
		return s, nil
	case ParamInteger:
		var n int64
		switch v := val.(type) {
		case int:
			n = int64(v)
		case int64:
			n = v
		case float64:
			n = int64(v)
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: parameter %q not integer", contracts.ErrInvalidArgument, f.Name)
			}
			n = parsed
		default:
			return nil, fmt.Errorf("%w: parameter %q not integer", contracts.ErrInvalidArgument, f.Name)
		}
		if f.Min != nil && float64(n) < *f.Min {
			return nil, fmt.Errorf("%w: parameter %q below min", contracts.ErrInvalidArgument, f.Name)
		}
		if f.Max != nil && float64(n) > *f.Max {
			return nil, fmt.Errorf("%w: parameter %q above max", contracts.ErrInvalidArgument, f.Name)
		}
		return n, nil
	case ParamNumber:
		var n float64
		switch v := val.(type) {
		case float64:
			n = v
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: parameter %q not number", contracts.ErrInvalidArgument, f.Name)
			}
			n = parsed
		default:
			return nil, fmt.Errorf("%w: parameter %q not number", contracts.ErrInvalidArgument, f.Name)
		}
		if f.Min != nil && n < *f.Min {
			return nil, fmt.Errorf("%w: parameter %q below min", contracts.ErrInvalidArgument, f.Name)
		}
		if f.Max != nil && n > *f.Max {
			return nil, fmt.Errorf("%w: parameter %q above max", contracts.ErrInvalidArgument, f.Name)
		}
		return n, nil
	case ParamBoolean:
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return nil, fmt.Errorf("%w: parameter %q not boolean", contracts.ErrInvalidArgument, f.Name)
		}
	case ParamArray:
		switch v := val.(type) {
		case []any:
			return v, nil
		case []string:
			out := make([]any, len(v))
			for i, s := range v {
				out[i] = s
			}
			return out, nil
		default:
			return nil, fmt.Errorf("%w: parameter %q not array", contracts.ErrInvalidArgument, f.Name)
		}
	default:
		return nil, fmt.Errorf("%w: unsupported type", contracts.ErrInvalidArgument)
	}
}

// ArgvMapping is a list of argv tokens. Tokens of the form {{name}} are substituted
// from validated parameters. Literal tokens must not contain shell compounds.
type ArgvMapping struct {
	Version contracts.SchemaVersion `json:"version"`
	Tokens  []string                `json:"tokens"`
}

// Validate checks argv mapping structure.
func (a ArgvMapping) Validate() error {
	v := a.Version
	if v == "" {
		v = SchemaV1
	}
	if v != SchemaV1 {
		return fmt.Errorf("%w: unsupported argv schema version %q", contracts.ErrInvalidArgument, a.Version)
	}
	for i, tok := range a.Tokens {
		if tok == "-c" || tok == "--command" || tok == "/c" {
			return fmt.Errorf("%w: argv token %d is a forbidden shell flag", contracts.ErrInvalidArgument, i)
		}
		if looksLikeShell(tok) && !isTemplate(tok) {
			return fmt.Errorf("%w: argv token %d looks like shell compound", contracts.ErrInvalidArgument, i)
		}
	}
	return nil
}

// Materialize builds argv from validated params. Result always passes ValidateArgv.
func (a ArgvMapping) Materialize(params map[string]any) ([]string, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	var out []string
	for i, tok := range a.Tokens {
		if isTemplate(tok) {
			name := strings.TrimSuffix(strings.TrimPrefix(tok, "{{"), "}}")
			name = strings.TrimSpace(name)
			if name == "binary" || name == "path" || name == "executable" {
				return nil, fmt.Errorf("%w: template cannot reference executable path", contracts.ErrInvalidArgument)
			}
			val, ok := params[name]
			if !ok {
				return nil, fmt.Errorf("%w: unknown template variable %q in token %d", contracts.ErrInvalidArgument, name, i)
			}
			s := fmt.Sprint(val)
			if looksLikeShell(s) {
				return nil, fmt.Errorf("%w: parameter %q value rejected by argv safety", contracts.ErrInvalidArgument, name)
			}
			out = append(out, s)
			continue
		}
		out = append(out, tok)
	}
	if err := process.ValidateArgv("tool", out); err != nil {
		return nil, err
	}
	return out, nil
}

func isTemplate(s string) bool {
	return strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}")
}

func looksLikeShell(s string) bool {
	if strings.Contains(s, "&&") || strings.Contains(s, "||") || strings.Contains(s, ";") {
		return true
	}
	if strings.Contains(s, "`") || strings.Contains(s, "$(") {
		return true
	}
	return false
}

// OutputField describes expected normalized output fields (data-only).
type OutputField struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // string, integer, boolean, object, array
}

// OutputSchema describes expected normalized output (documentation + light validation).
type OutputSchema struct {
	Version contracts.SchemaVersion `json:"version"`
	Fields  []OutputField           `json:"fields,omitempty"`
}

// Validate checks output schema structure. Does not execute anything.
func (o OutputSchema) Validate() error {
	v := o.Version
	if v == "" {
		v = SchemaV1
	}
	if v != SchemaV1 {
		return fmt.Errorf("%w: unsupported output schema version %q", contracts.ErrInvalidArgument, o.Version)
	}
	for _, f := range o.Fields {
		if f.Name == "" {
			return fmt.Errorf("%w: output field name required", contracts.ErrInvalidArgument)
		}
	}
	return nil
}

// ToolSchemas groups optional declarative schemas on a tool.
type ToolSchemas struct {
	Parameters *ParameterSchema `json:"parameters,omitempty"`
	Argv       *ArgvMapping     `json:"argv,omitempty"`
	Output     *OutputSchema    `json:"output,omitempty"`
}

// Validate validates all present schemas.
func (t ToolSchemas) Validate() error {
	if t.Parameters != nil {
		if err := t.Parameters.Validate(); err != nil {
			return err
		}
	}
	if t.Argv != nil {
		if err := t.Argv.Validate(); err != nil {
			return err
		}
	}
	if t.Output != nil {
		if err := t.Output.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// BuildArgs validates params and materializes argv when Argv mapping is present.
func (t ToolSchemas) BuildArgs(params map[string]any) ([]string, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	normalized := params
	var err error
	if t.Parameters != nil {
		normalized, err = t.Parameters.Apply(params)
		if err != nil {
			return nil, err
		}
	}
	if t.Argv != nil {
		return t.Argv.Materialize(normalized)
	}
	// Fall back: pass through string args list if present (same as defaultBuildArgs subset)
	return nil, fmt.Errorf("%w: no argv mapping declared", contracts.ErrInvalidArgument)
}
