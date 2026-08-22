package tools

import (
	"context"
	"fmt"
	"sync"

	"cyberos/internal/contracts"
	"cyberos/internal/execution"
)

// Platform is the extensible security-tool platform.
// New tools are added by registering Adapters + Manifests — not by changing core packages.
type Platform struct {
	mu      sync.RWMutex
	manager *Manager
	packs   *PackRegistry
	catalog *Catalog // optional discovery catalog from pack loader (Phase 11C)
	// status cache from last health sweep
	status map[string]Status
}

// NewPlatform constructs a tool platform around a Manager.
func NewPlatform(mgr *Manager) *Platform {
	if mgr == nil {
		mgr = NewManager(nil)
	}
	return &Platform{
		manager: mgr,
		packs:   NewPackRegistry(),
		catalog: NewCatalog(),
		status:  make(map[string]Status),
	}
}

// Catalog returns the discovery-only tool catalog.
func (p *Platform) Catalog() *Catalog {
	if p == nil {
		return nil
	}
	return p.catalog
}

// Manager returns the underlying adapter manager.
func (p *Platform) Manager() *Manager { return p.manager }

// Packs returns the pack registry.
func (p *Platform) Packs() *PackRegistry { return p.packs }

// RegisterAdapter validates and registers a tool adapter (discovery only).
func (p *Platform) RegisterAdapter(a Adapter) error {
	if p == nil {
		return contracts.ErrInvalidArgument
	}
	d := a.Descriptor()
	m := Manifest{
		ID: d.ID, Name: d.Name, Version: d.Version, Runtime: d.Runtime,
		Executable: d.Binary, Capabilities: d.Capabilities, ExecutorID: d.ExecutorID,
		Description: d.Description,
	}
	if err := m.Validate(); err != nil {
		return err
	}
	return p.manager.Register(a)
}

// Discover lists registered tool descriptors (not authorization).
func (p *Platform) Discover() []Descriptor {
	if p == nil || p.manager == nil {
		return nil
	}
	return p.manager.List()
}

// Resolve returns an adapter by tool id (fail closed if unknown).
func (p *Platform) Resolve(toolID string) (Adapter, error) {
	return p.manager.Get(toolID)
}

// HealthSweep runs health checks and updates availability snapshots.
func (p *Platform) HealthSweep(ctx context.Context) map[string]Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]Status)
	for _, d := range p.manager.List() {
		a, err := p.manager.Get(d.ID)
		st := Status{
			Manifest: Manifest{
				ID: d.ID, Name: d.Name, Version: d.Version, Runtime: d.Runtime,
				Executable: d.Binary, Capabilities: d.Capabilities, ExecutorID: d.ExecutorID,
				Description: d.Description,
			},
			Registered:   true,
			Availability: AvailabilityUnknown,
		}
		if err != nil {
			st.Availability = AvailabilityMissing
			out[d.ID] = st
			continue
		}
		h := a.Health(ctx)
		st.Health = h
		switch h.Status {
		case HealthHealthy:
			st.Availability = AvailabilityInstalled
		case HealthDegraded:
			st.Availability = AvailabilityInstalled // present but limited
		case HealthUnhealthy:
			if contains(h.Message, "not found") {
				st.Availability = AvailabilityMissing
			} else {
				st.Availability = AvailabilityUnhealthy
			}
		default:
			st.Availability = AvailabilityUnknown
		}
		out[d.ID] = st
	}
	p.status = out
	return out
}

// StatusOf returns last known status for a tool.
func (p *Platform) StatusOf(toolID string) (Status, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.status[toolID]
	return s, ok
}

// WireExecutors registers all adapters into an execution registry for the Broker router.
func (p *Platform) WireExecutors(reg *execution.Registry) error {
	if p == nil || reg == nil {
		return fmt.Errorf("%w: platform/registry not configured", contracts.ErrInvalidArgument)
	}
	return p.manager.RegisterAllExecutors(reg)
}

// RegisterPack stores a pack and optionally associates tool IDs already registered.
func (p *Platform) RegisterPack(pack Pack) error {
	return p.packs.Register(pack)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// CapabilityMapping describes Pack → Tool → Capability (discovery only).
type CapabilityMapping struct {
	PackID       string                  `json:"pack_id,omitempty"`
	ToolID       string                  `json:"tool_id"`
	ExecutorID   string                  `json:"executor_id,omitempty"`
	CapabilityID contracts.CapabilityID  `json:"capability_id"`
	Version      contracts.SchemaVersion `json:"version,omitempty"`
}

// CapabilityMap builds discovery mappings from registered packs and adapters.
// It does not authorize execution.
func (p *Platform) CapabilityMap() []CapabilityMapping {
	if p == nil || p.manager == nil {
		return nil
	}
	toolToPack := map[string]string{}
	packVer := map[string]contracts.SchemaVersion{}
	for _, pack := range p.packs.List() {
		packVer[pack.ID] = pack.Version
		for _, tid := range pack.ToolIDs {
			toolToPack[tid] = pack.ID
		}
	}
	var out []CapabilityMapping
	seen := map[string]struct{}{}
	add := func(d Descriptor) {
		pid := toolToPack[d.ID]
		for _, cap := range d.Capabilities {
			key := d.ID + ":" + string(cap)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, CapabilityMapping{
				PackID: pid, ToolID: d.ID, ExecutorID: d.ExecutorID,
				CapabilityID: cap, Version: packVer[pid],
			})
		}
	}
	if p.manager != nil {
		for _, d := range p.manager.List() {
			add(d)
		}
	}
	if p.catalog != nil {
		for _, d := range p.catalog.List() {
			add(d)
		}
	}
	return out
}

// LookupCapability finds tools declaring a capability (discovery only).
func (p *Platform) LookupCapability(cap contracts.CapabilityID) []CapabilityMapping {
	var out []CapabilityMapping
	for _, m := range p.CapabilityMap() {
		if m.CapabilityID == cap {
			out = append(out, m)
		}
	}
	return out
}
