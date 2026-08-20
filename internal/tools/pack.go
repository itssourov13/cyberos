package tools

import (
	"fmt"
	"sync"

	"cyberos/internal/contracts"
)

// Pack groups related tools for future catalogs (Recon, Web, Vuln, Network).
// Pack membership is metadata only — not authorization.
type Pack struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	ToolIDs     []string `json:"tool_ids"`
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

// Register adds or replaces a pack definition.
func (r *PackRegistry) Register(p Pack) error {
	if r == nil {
		return contracts.ErrInvalidArgument
	}
	if p.ID == "" || p.Name == "" {
		return fmt.Errorf("%w: pack id and name required", contracts.ErrInvalidArgument)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := p
	cp.ToolIDs = append([]string{}, p.ToolIDs...)
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

// Standard pack IDs for future expansion.
const (
	PackRecon = "pack.recon"
	PackWeb   = "pack.web"
	PackVuln  = "pack.vuln"
	PackNet   = "pack.network"
)
