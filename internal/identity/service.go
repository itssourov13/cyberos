// Package identity provides principal management services.
// It sits above the domain and storage ports and enforces
// identity-related invariants before any authorization work.
package identity

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/principal"
	"cyberos/internal/storage/ports"
)

// Service manages principals (identity).
type Service struct {
	repo ports.PrincipalRepository
}

// NewService constructs an identity service.
func NewService(repo ports.PrincipalRepository) *Service {
	return &Service{repo: repo}
}

// Create registers a new principal after validation.
func (s *Service) Create(ctx context.Context, p *principal.Principal) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("%w: identity service not configured", contracts.ErrInvalidArgument)
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if !p.IsActive() {
		return fmt.Errorf("%w: cannot create inactive principal", contracts.ErrInvalidArgument)
	}
	return s.repo.Create(ctx, p)
}

// Get returns a principal by ID.
func (s *Service) Get(ctx context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: identity service not configured", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("principal.id", string(id)); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// RequireActive loads the principal and ensures it is active and valid.
// This is the gate used before any authorization decision.
func (s *Service) RequireActive(ctx context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if !p.IsActive() {
		return nil, fmt.Errorf("%w: principal %s is not active", contracts.ErrForbidden, id)
	}
	return p, nil
}

// List returns principals with basic pagination.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*principal.Principal, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: identity service not configured", contracts.ErrInvalidArgument)
	}
	return s.repo.List(ctx, limit, offset)
}
