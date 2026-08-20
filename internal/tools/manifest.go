package tools

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Availability is installation/readiness status for a tool.
type Availability string

const (
	AvailabilityUnknown      Availability = "UNKNOWN"
	AvailabilityInstalled    Availability = "INSTALLED"
	AvailabilityMissing      Availability = "MISSING"
	AvailabilityWrongVersion Availability = "WRONG_VERSION"
	AvailabilityUnhealthy    Availability = "UNHEALTHY"
	AvailabilityUnsupported  Availability = "UNSUPPORTED"
	AvailabilityDisabled     Availability = "DISABLED"
)

// Valid reports whether the availability value is known.
func (a Availability) Valid() bool {
	switch a {
	case AvailabilityUnknown, AvailabilityInstalled, AvailabilityMissing,
		AvailabilityWrongVersion, AvailabilityUnhealthy, AvailabilityUnsupported, AvailabilityDisabled:
		return true
	default:
		return false
	}
}

// Manifest is the extensible tool declaration used by the platform.
// Adding a tool should primarily mean providing a Manifest + Adapter — not core changes.
type Manifest struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Version      contracts.SchemaVersion  `json:"version"`
	Runtime      contracts.PluginRuntime  `json:"runtime"`
	Executable   string                   `json:"executable,omitempty"`
	Capabilities []contracts.CapabilityID `json:"capabilities"`
	ExecutorID   string                   `json:"executor_id"`
	Description  string                   `json:"description,omitempty"`
	PackID       string                   `json:"pack_id,omitempty"`
	Platforms    []string                 `json:"platforms,omitempty"` // e.g. linux, darwin
	// Optional operational hints (not authorization)
	DefaultTimeout time.Duration `json:"default_timeout,omitempty"`
	MaxOutputBytes int64         `json:"max_output_bytes,omitempty"`
	ArgSchema      []string      `json:"arg_schema,omitempty"` // documented keys only
	OutputFormat   string        `json:"output_format,omitempty"`
}

// Validate checks required manifest fields.
func (m Manifest) Validate() error {
	if err := contracts.ValidateNonEmpty("tool.id", m.ID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("tool.name", m.Name); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("tool.executor_id", m.ExecutorID); err != nil {
		return err
	}
	if !m.Runtime.Valid() {
		return fmt.Errorf("%w: invalid runtime", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("tool.version", string(m.Version)); err != nil {
		return err
	}
	return nil
}

// Descriptor converts to the lightweight Descriptor used by adapters.
func (m Manifest) Descriptor() Descriptor {
	return Descriptor{
		ID: m.ID, Name: m.Name, Version: m.Version, Runtime: m.Runtime,
		Capabilities: m.Capabilities, ExecutorID: m.ExecutorID,
		Description: m.Description, Binary: m.Executable,
	}
}

// ToolManifest maps to contracts.ToolManifest for executor registration.
func (m Manifest) ToolManifest() contracts.ToolManifest {
	return m.Descriptor().ToolManifest()
}

// Status is discovery/health snapshot (never authorization).
type Status struct {
	Manifest     Manifest     `json:"manifest"`
	Availability Availability `json:"availability"`
	Health       HealthReport `json:"health"`
	Registered   bool         `json:"registered"`
}
