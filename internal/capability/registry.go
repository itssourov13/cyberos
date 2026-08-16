// Package capability provides the Capability Registry.
//
// The registry answers: "What is technically available?"
// It does NOT answer authorization questions.
// Discovery of a capability MUST NEVER imply permission to use it.
package capability

import (
	"context"
	"fmt"
	"sync"

	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/storage/ports"
)

// Registry is the capability discovery service.
type Registry struct {
	repo ports.CapabilityRepository
	// in-memory fallback when repo is nil (tests / early bootstrap)
	mu    sync.RWMutex
	local map[contracts.CapabilityID]*domaincap.Capability
}

// NewRegistry constructs a registry backed by a repository.
// repo may be nil for pure in-memory use (tests).
func NewRegistry(repo ports.CapabilityRepository) *Registry {
	return &Registry{
		repo:  repo,
		local: make(map[contracts.CapabilityID]*domaincap.Capability),
	}
}

// Register adds a capability to the registry after validation.
// Registration is discovery only — it grants no permissions.
func (r *Registry) Register(ctx context.Context, c *domaincap.Capability) error {
	if r == nil {
		return fmt.Errorf("%w: registry not configured", contracts.ErrInvalidArgument)
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.IsActive() {
		return fmt.Errorf("%w: cannot register inactive capability", contracts.ErrInvalidArgument)
	}
	if r.repo != nil {
		return r.repo.Create(ctx, c)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.local[c.ID]; exists {
		return contracts.ErrConflict
	}
	cp := *c
	r.local[c.ID] = &cp
	return nil
}

// Get returns a capability by ID if present.
func (r *Registry) Get(ctx context.Context, id contracts.CapabilityID) (*domaincap.Capability, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: registry not configured", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("capability.id", string(id)); err != nil {
		return nil, err
	}
	if r.repo != nil {
		return r.repo.GetByID(ctx, id)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.local[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

// RequireKnown ensures the capability exists and is active.
// This is discovery only — callers must still pass through the Control Plane
// for authorization.
func (r *Registry) RequireKnown(ctx context.Context, id contracts.CapabilityID) (*domaincap.Capability, error) {
	c, err := r.Get(ctx, id)
	if err != nil {
		if err == contracts.ErrNotFound {
			return nil, fmt.Errorf("%w: %s", contracts.ErrCapabilityUnknown, id)
		}
		return nil, err
	}
	if !c.IsActive() {
		return nil, fmt.Errorf("%w: capability %s is inactive", contracts.ErrCapabilityUnknown, id)
	}
	return c, nil
}

// List returns registered capabilities.
func (r *Registry) List(ctx context.Context, limit, offset int) ([]*domaincap.Capability, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: registry not configured", contracts.ErrInvalidArgument)
	}
	if r.repo != nil {
		return r.repo.List(ctx, limit, offset)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domaincap.Capability
	i := 0
	for _, c := range r.local {
		if i < offset {
			i++
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		cp := *c
		out = append(out, &cp)
		i++
	}
	return out, nil
}

// Descriptor returns the contract-level descriptor for discovery APIs.
// Presence in the registry never implies authorization.
func (r *Registry) Descriptor(ctx context.Context, id contracts.CapabilityID) (contracts.CapabilityDescriptor, error) {
	c, err := r.Get(ctx, id)
	if err != nil {
		return contracts.CapabilityDescriptor{}, err
	}
	return c.Descriptor(), nil
}
