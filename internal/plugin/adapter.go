// Package plugin defines the adapter architecture for future tool integrations.
//
// Adapters declare capabilities and may prepare work, but they must NEVER
// execute tools directly. All execution goes through Tool Broker → Sandbox.
package plugin

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
)

// Adapter is the secure boundary for a tool integration.
// Implementations must not invoke shell or real offensive tools in Phase 3.
type Adapter interface {
	// Manifest returns the static description of this adapter.
	Manifest() contracts.PluginManifest
	// Validate checks adapter configuration and declared capabilities.
	Validate(ctx context.Context) error
	// Prepare may return opaque parameters for a future ExecutionRequest.
	// It must not perform side-effecting tool work.
	Prepare(ctx context.Context, capabilityID contracts.CapabilityID, input map[string]any) (map[string]any, error)
}

// BaseAdapter is a minimal in-process adapter used for tests and placeholders.
type BaseAdapter struct {
	manifest contracts.PluginManifest
}

// NewBaseAdapter constructs a validated base adapter.
func NewBaseAdapter(m contracts.PluginManifest) (*BaseAdapter, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &BaseAdapter{manifest: m}, nil
}

// Manifest implements Adapter.
func (a *BaseAdapter) Manifest() contracts.PluginManifest { return a.manifest }

// Validate implements Adapter.
func (a *BaseAdapter) Validate(ctx context.Context) error {
	_ = ctx
	return a.manifest.Validate()
}

// Prepare implements Adapter — returns input copy only; no side effects.
func (a *BaseAdapter) Prepare(ctx context.Context, capabilityID contracts.CapabilityID, input map[string]any) (map[string]any, error) {
	_ = ctx
	found := false
	for _, c := range a.manifest.Capabilities {
		if c == capabilityID {
			found = true
			break
		}
	}
	if !found && len(a.manifest.Capabilities) > 0 {
		return nil, fmt.Errorf("%w: capability %s not declared by plugin %s",
			contracts.ErrCapabilityUnknown, capabilityID, a.manifest.ID)
	}
	out := map[string]any{}
	for k, v := range input {
		out[k] = v
	}
	out["plugin_id"] = a.manifest.ID
	out["capability_id"] = string(capabilityID)
	return out, nil
}

// Ensure interface compliance.
var _ Adapter = (*BaseAdapter)(nil)
