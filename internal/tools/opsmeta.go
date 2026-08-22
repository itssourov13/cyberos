package tools

import (
	"fmt"
	"strings"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/secrets"
)

// RuntimeMeta describes required runtime (discovery/ops only — not authorization).
type RuntimeMeta struct {
	Type              contracts.PluginRuntime `json:"type,omitempty"`
	Binary            string                  `json:"binary,omitempty"`
	VersionConstraint string                  `json:"version_constraint,omitempty"` // descriptive, e.g. ">=7"
}

// Validate checks runtime metadata. Does not probe host or execute binaries.
func (r RuntimeMeta) Validate() error {
	if r.Type != "" && !r.Type.Valid() {
		return fmt.Errorf("%w: invalid runtime type %q", contracts.ErrInvalidArgument, r.Type)
	}
	if r.Binary != "" {
		if strings.ContainsAny(r.Binary, " \t\n|&;<>$`\\\"'") {
			return fmt.Errorf("%w: runtime binary contains unsafe characters", contracts.ErrInvalidArgument)
		}
		if strings.Contains(r.Binary, "..") {
			return fmt.Errorf("%w: runtime binary path traversal rejected", contracts.ErrInvalidArgument)
		}
	}
	if strings.ContainsAny(r.VersionConstraint, "\n\r;&|`$") {
		return fmt.Errorf("%w: invalid version constraint", contracts.ErrInvalidArgument)
	}
	return nil
}

// DependencyMeta declares a dependency on another pack or tool (metadata only).
type DependencyMeta struct {
	PackID   string                  `json:"pack_id,omitempty"`
	ToolID   string                  `json:"tool_id,omitempty"`
	Version  contracts.SchemaVersion `json:"version,omitempty"` // exact version if set
	Optional bool                    `json:"optional,omitempty"`
}

// Validate checks dependency declaration shape.
func (d DependencyMeta) Validate() error {
	if d.PackID == "" && d.ToolID == "" {
		return fmt.Errorf("%w: dependency requires pack_id or tool_id", contracts.ErrInvalidArgument)
	}
	if strings.ContainsAny(d.PackID+d.ToolID, " \t\n;&|") {
		return fmt.Errorf("%w: invalid dependency id", contracts.ErrInvalidArgument)
	}
	return nil
}

// ConfigRef is a declarative configuration key reference (never a secret value).
// Name must be a simple identifier; no path traversal or env expansion in manifests.
type ConfigRef struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
}

// Validate checks config reference shape.
func (c ConfigRef) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: config ref name required", contracts.ErrInvalidArgument)
	}
	for _, r := range c.Name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("%w: config ref name has invalid characters", contracts.ErrInvalidArgument)
	}
	if strings.Contains(c.Name, "..") {
		return fmt.Errorf("%w: config ref path traversal rejected", contracts.ErrInvalidArgument)
	}
	return nil
}

// SecretRefMeta embeds secrets.Ref for pack/tool manifests (opaque reference only).
type SecretRefMeta struct {
	Name string      `json:"name"` // logical name
	Ref  secrets.Ref `json:"ref"`  // e.g. env:CYBEROS_TOKEN — never the value
}

// Validate ensures secret material is not embedded.
func (s SecretRefMeta) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("%w: secret ref name required", contracts.ErrInvalidArgument)
	}
	if err := secrets.ValidateRef(s.Ref); err != nil {
		return err
	}
	// Reject refs that look like they contain the secret inline
	v := string(s.Ref)
	if strings.Contains(strings.ToLower(v), "password=") || len(v) > 256 {
		return fmt.Errorf("%w: invalid secret ref", contracts.ErrInvalidArgument)
	}
	return nil
}

// ResourceHints are soft requests; Sandbox/process.Policy enforce actual limits.
type ResourceHints struct {
	TimeoutSeconds int   `json:"timeout_seconds,omitempty"`
	MaxOutputBytes int64 `json:"max_output_bytes,omitempty"`
}

// Validate rejects absurd values that would try to disable limits (hints only).
func (r ResourceHints) Validate() error {
	if r.TimeoutSeconds < 0 || r.TimeoutSeconds > 3600 {
		return fmt.Errorf("%w: timeout_seconds out of allowed hint range 0..3600", contracts.ErrInvalidArgument)
	}
	if r.MaxOutputBytes < 0 || r.MaxOutputBytes > 64*1024*1024 {
		return fmt.Errorf("%w: max_output_bytes out of allowed hint range", contracts.ErrInvalidArgument)
	}
	return nil
}

// HealthMeta is declarative health expectation (not executed at install).
type HealthMeta struct {
	VersionArgs []string `json:"version_args,omitempty"`
}

// Validate checks health metadata without running processes.
func (h HealthMeta) Validate() error {
	for _, a := range h.VersionArgs {
		if a == "-c" || a == "--command" || a == "/c" {
			return fmt.Errorf("%w: health args cannot use shell flags", contracts.ErrInvalidArgument)
		}
		if strings.ContainsAny(a, ";&|`$") {
			return fmt.Errorf("%w: unsafe health arg", contracts.ErrInvalidArgument)
		}
	}
	return nil
}

// OpsMeta aggregates optional operational metadata for packs/tools (Phase 11D).
type OpsMeta struct {
	Runtime      *RuntimeMeta     `json:"runtime,omitempty"`
	Dependencies []DependencyMeta `json:"dependencies,omitempty"`
	ConfigRefs   []ConfigRef      `json:"config_refs,omitempty"`
	SecretRefs   []SecretRefMeta  `json:"secret_refs,omitempty"`
	Resources    *ResourceHints   `json:"resources,omitempty"`
	Health       *HealthMeta      `json:"health,omitempty"`
}

// Validate validates all ops metadata without executing or resolving secrets.
func (o OpsMeta) Validate() error {
	if o.Runtime != nil {
		if err := o.Runtime.Validate(); err != nil {
			return err
		}
	}
	seen := map[string]struct{}{}
	for i, d := range o.Dependencies {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("dependencies[%d]: %w", i, err)
		}
		key := d.PackID + "|" + d.ToolID + "|" + string(d.Version)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate dependency %s", contracts.ErrInvalidArgument, key)
		}
		seen[key] = struct{}{}
	}
	for i, c := range o.ConfigRefs {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("config_refs[%d]: %w", i, err)
		}
	}
	for i, s := range o.SecretRefs {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("secret_refs[%d]: %w", i, err)
		}
	}
	if o.Resources != nil {
		if err := o.Resources.Validate(); err != nil {
			return err
		}
	}
	if o.Health != nil {
		if err := o.Health.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CheckDependencies ensures required deps exist in the platform (fail closed).
// Does not download or install anything.
func CheckDependencies(plat *Platform, deps []DependencyMeta) error {
	if len(deps) == 0 {
		return nil
	}
	if plat == nil {
		return fmt.Errorf("%w: platform required for dependency check", contracts.ErrInvalidArgument)
	}
	packs := map[string]Pack{}
	for _, p := range plat.Packs().List() {
		packs[p.ID] = p
	}
	for _, d := range deps {
		if d.Optional {
			continue
		}
		if d.PackID != "" {
			p, ok := packs[d.PackID]
			if !ok {
				return fmt.Errorf("%w: missing required pack dependency %s", contracts.ErrNotFound, d.PackID)
			}
			if d.Version != "" && p.Version != d.Version {
				return fmt.Errorf("%w: pack %s version conflict want %s have %s",
					contracts.ErrConflict, d.PackID, d.Version, p.Version)
			}
		}
		if d.ToolID != "" {
			if plat.Catalog() != nil {
				td, err := plat.Catalog().Get(d.ToolID)
				if err != nil {
					return fmt.Errorf("%w: missing required tool dependency %s", contracts.ErrNotFound, d.ToolID)
				}
				if d.Version != "" && td.Version != d.Version {
					return fmt.Errorf("%w: tool %s version conflict", contracts.ErrConflict, d.ToolID)
				}
			}
		}
	}
	return nil
}

// EffectiveTimeoutHint returns a duration hint capped for safety (not enforcement).
func (r ResourceHints) EffectiveTimeoutHint() time.Duration {
	if r.TimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(r.TimeoutSeconds) * time.Second
}
