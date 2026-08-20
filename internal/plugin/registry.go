package plugin

import (
	"context"
	"fmt"
	"sync"

	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
)

// Registration is a plugin entry with lifecycle state.
type Registration struct {
	Manifest contracts.PluginManifest
	State    contracts.PluginState
	Adapter  Adapter
}

// Registry manages plugin discovery and lifecycle.
// Discovery ≠ authorization.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*Registration
	capReg  *capability.Registry // optional: declare capabilities on activate
}

// NewRegistry constructs an empty plugin registry.
// capReg may be nil; when set, Activate registers declared capabilities for discovery.
func NewRegistry(capReg *capability.Registry) *Registry {
	return &Registry{
		plugins: make(map[string]*Registration),
		capReg:  capReg,
	}
}

// Register validates and stores an adapter in REGISTERED state.
func (r *Registry) Register(ctx context.Context, a Adapter) error {
	if r == nil {
		return fmt.Errorf("%w: plugin registry not configured", contracts.ErrInvalidArgument)
	}
	if a == nil {
		return fmt.Errorf("%w: adapter is nil", contracts.ErrInvalidArgument)
	}
	m := a.Manifest()
	if err := m.Validate(); err != nil {
		return err
	}
	if err := a.Validate(ctx); err != nil {
		return fmt.Errorf("%w: adapter validate: %v", contracts.ErrInvalidArgument, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[m.ID]; exists {
		return contracts.ErrConflict
	}
	r.plugins[m.ID] = &Registration{
		Manifest: m,
		State:    contracts.PluginStateRegistered,
		Adapter:  a,
	}
	return nil
}

// Activate moves REGISTERED → ACTIVE and optionally registers capabilities.
func (r *Registry) Activate(ctx context.Context, pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	reg, ok := r.plugins[pluginID]
	if !ok {
		return contracts.ErrNotFound
	}
	if reg.State == contracts.PluginStateDisabled {
		return fmt.Errorf("%w: plugin disabled", contracts.ErrInvalidArgument)
	}
	if r.capReg != nil {
		for _, cid := range reg.Manifest.Capabilities {
			// Best-effort discovery registration; ignore conflict if already present.
			c, err := domaincap.New(cid, string(cid), contracts.CapabilityTypeCustom)
			if err != nil {
				continue
			}
			_ = r.capReg.Register(ctx, c)
		}
	}
	reg.State = contracts.PluginStateActive
	return nil
}

// Disable moves a plugin to DISABLED.
func (r *Registry) Disable(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	reg, ok := r.plugins[pluginID]
	if !ok {
		return contracts.ErrNotFound
	}
	reg.State = contracts.PluginStateDisabled
	return nil
}

// Get returns a registration by ID.
func (r *Registry) Get(pluginID string) (Registration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.plugins[pluginID]
	if !ok {
		return Registration{}, contracts.ErrNotFound
	}
	return *reg, nil
}

// List returns all registrations.
func (r *Registry) List() []Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Registration, 0, len(r.plugins))
	for _, reg := range r.plugins {
		out = append(out, *reg)
	}
	return out
}

// ListActive returns ACTIVE plugins only.
func (r *Registry) ListActive() []Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Registration
	for _, reg := range r.plugins {
		if reg.State == contracts.PluginStateActive {
			out = append(out, *reg)
		}
	}
	return out
}

// Prepare routes to an active adapter's Prepare. Does not execute tools.
func (r *Registry) Prepare(ctx context.Context, pluginID string, capabilityID contracts.CapabilityID, input map[string]any) (map[string]any, error) {
	r.mu.RLock()
	reg, ok := r.plugins[pluginID]
	r.mu.RUnlock()
	if !ok {
		return nil, contracts.ErrNotFound
	}
	if reg.State != contracts.PluginStateActive {
		return nil, fmt.Errorf("%w: plugin not active", contracts.ErrInvalidArgument)
	}
	return reg.Adapter.Prepare(ctx, capabilityID, input)
}
