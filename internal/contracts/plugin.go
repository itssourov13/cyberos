package contracts

// PluginRuntime identifies the language/runtime boundary for an adapter.
type PluginRuntime string

const (
	PluginRuntimeGo     PluginRuntime = "go"
	PluginRuntimePython PluginRuntime = "python"
	PluginRuntimeRust   PluginRuntime = "rust"
)

// Valid reports whether the runtime is known.
func (r PluginRuntime) Valid() bool {
	switch r {
	case PluginRuntimeGo, PluginRuntimePython, PluginRuntimeRust:
		return true
	default:
		return false
	}
}

// PluginState is the lifecycle state of a registered plugin/adapter.
type PluginState string

const (
	PluginStateRegistered PluginState = "REGISTERED"
	PluginStateActive     PluginState = "ACTIVE"
	PluginStateDisabled   PluginState = "DISABLED"
	PluginStateFailed     PluginState = "FAILED"
)

// Valid reports whether the state is known.
func (s PluginState) Valid() bool {
	switch s {
	case PluginStateRegistered, PluginStateActive, PluginStateDisabled, PluginStateFailed:
		return true
	default:
		return false
	}
}

// PluginManifest describes a plugin/adapter for discovery.
// Discovery never implies authorization to execute its capabilities.
type PluginManifest struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Version      SchemaVersion  `json:"version"`
	Runtime      PluginRuntime  `json:"runtime"`
	Description  string         `json:"description,omitempty"`
	Capabilities []CapabilityID `json:"capabilities,omitempty"`
	// ExecutorHint is a soft label for grant constraints (e.g. "mock", "sandbox").
	ExecutorHint string `json:"executor_hint,omitempty"`
}

// Validate checks required manifest fields.
func (m PluginManifest) Validate() error {
	if err := ValidateNonEmpty("plugin.id", m.ID); err != nil {
		return err
	}
	if err := ValidateNonEmpty("plugin.name", m.Name); err != nil {
		return err
	}
	if !m.Runtime.Valid() {
		return ErrInvalidArgument
	}
	if err := ValidateNonEmpty("plugin.version", string(m.Version)); err != nil {
		return err
	}
	return nil
}
