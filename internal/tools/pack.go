package tools

import (
	"fmt"
	"sync"

	"cyberos/internal/contracts"
)

// PackState is the lifecycle state of a tool pack (metadata only — not authorization).
type PackState string

const (
	PackStateRegistered PackState = "REGISTERED"
	PackStateActive     PackState = "ACTIVE"
	PackStateDisabled   PackState = "DISABLED"
	PackStateDeprecated PackState = "DEPRECATED"
)

// Valid reports whether the pack state is known.
func (s PackState) Valid() bool {
	switch s {
	case PackStateRegistered, PackStateActive, PackStateDisabled, PackStateDeprecated:
		return true
	default:
		return false
	}
}

// Pack groups related tools for catalogs (Recon, Web, Vuln, Network).
// Pack membership and version are metadata only — not authorization.
type Pack struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Version       contracts.SchemaVersion  `json:"version"`
	State         PackState                `json:"state"`
	Description   string                   `json:"description,omitempty"`
	ToolIDs       []string                 `json:"tool_ids"`
	CapabilityIDs []contracts.CapabilityID `json:"capability_ids,omitempty"`
	// ContractVersion versions the pack extension contract (not the pack product version).
	// Empty defaults to ExtensionContractV1 for backward compatibility.
	ContractVersion contracts.SchemaVersion `json:"contract_version,omitempty"`
	// Ops is optional runtime/dependency/config/secret metadata (Phase 11D).
	Ops *OpsMeta `json:"ops,omitempty"`
}

// Validate checks pack identity, version, and state.
func (p Pack) Validate() error {
	if err := contracts.ValidateNonEmpty("pack.id", p.ID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("pack.name", p.Name); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("pack.version", string(p.Version)); err != nil {
		return err
	}
	if p.State != "" && !p.State.Valid() {
		return fmt.Errorf("%w: invalid pack state %q", contracts.ErrInvalidArgument, p.State)
	}
	if err := contracts.ValidateExtensionContract(p.ContractVersion); err != nil {
		return err
	}
	if p.Ops != nil {
		if err := p.Ops.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// PackRegistry holds named packs.
type PackRegistry struct {
	mu    sync.RWMutex
	packs map[string]Pack
}

// NewPackRegistry constructs an empty pack registry.
func NewPackRegistry() *PackRegistry {
	return &PackRegistry{packs: make(map[string]Pack)}
}

// Register adds a pack definition. Duplicate ID with a different version is rejected.
// Same ID + same version is idempotent. Discovery only — not authorization.
func (r *PackRegistry) Register(p Pack) error {
	if r == nil {
		return contracts.ErrInvalidArgument
	}
	if p.State == "" {
		p.State = PackStateRegistered
	}
	if p.Version == "" {
		p.Version = "1.0.0"
	}
	if p.ContractVersion == "" {
		p.ContractVersion = contracts.ExtensionContractV1
	}
	if err := p.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.packs[p.ID]; ok {
		if existing.Version != p.Version {
			return fmt.Errorf("%w: pack %s already registered at version %s (got %s)",
				contracts.ErrConflict, p.ID, existing.Version, p.Version)
		}
		// idempotent same version — refresh metadata
	}
	cp := p
	cp.ToolIDs = append([]string{}, p.ToolIDs...)
	cp.CapabilityIDs = append([]contracts.CapabilityID{}, p.CapabilityIDs...)
	r.packs[p.ID] = cp
	return nil
}

// Get returns a pack by id.
func (r *PackRegistry) Get(id string) (Pack, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.packs[id]
	if !ok {
		return Pack{}, contracts.ErrNotFound
	}
	return p, nil
}

// List returns all packs.
func (r *PackRegistry) List() []Pack {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Pack, 0, len(r.packs))
	for _, p := range r.packs {
		out = append(out, p)
	}
	return out
}

// ListActive returns packs in ACTIVE or REGISTERED state (discoverable).
func (r *PackRegistry) ListActive() []Pack {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Pack, 0, len(r.packs))
	for _, p := range r.packs {
		if p.State == PackStateActive || p.State == PackStateRegistered {
			out = append(out, p)
		}
	}
	return out
}

// SetState updates pack lifecycle state (metadata only).
func (r *PackRegistry) SetState(id string, state PackState) error {
	if !state.Valid() {
		return fmt.Errorf("%w: invalid pack state", contracts.ErrInvalidArgument)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.packs[id]
	if !ok {
		return contracts.ErrNotFound
	}
	p.State = state
	r.packs[id] = p
	return nil
}

// Standard pack IDs for expansion.
const (
	PackRecon = "pack.recon"
	PackWeb   = "pack.web"
	PackVuln  = "pack.vuln"
	PackNet   = "pack.network"
)
