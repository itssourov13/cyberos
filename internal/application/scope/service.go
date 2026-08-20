// Package scope provides application services that treat Scope as the
// mandatory authorization boundary.
package scope

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/scope"
	"cyberos/internal/storage/ports"
)

// Service manages scopes and enforces boundary invariants.
type Service struct {
	repo ports.ScopeRepository
}

// NewService constructs a scope application service.
func NewService(repo ports.ScopeRepository) *Service {
	return &Service{repo: repo}
}

// Create persists a new scope after validation.
func (s *Service) Create(ctx context.Context, sc *scope.Scope) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("%w: scope service not configured", contracts.ErrInvalidArgument)
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	return s.repo.Create(ctx, sc)
}

// Get returns a scope by ID.
func (s *Service) Get(ctx context.Context, id contracts.ScopeID) (*scope.Scope, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: scope service not configured", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("scope.id", string(id)); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// Update persists scope changes after validation.
func (s *Service) Update(ctx context.Context, sc *scope.Scope) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("%w: scope service not configured", contracts.ErrInvalidArgument)
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, sc)
}

// ListByProject returns scopes for a project.
func (s *Service) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*scope.Scope, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: scope service not configured", contracts.ErrInvalidArgument)
	}
	return s.repo.ListByProject(ctx, projectID, limit, offset)
}

// ListAll returns scopes across projects (durable catalog support).
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*scope.Scope, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: scope service not configured", contracts.ErrInvalidArgument)
	}
	return s.repo.ListAll(ctx, limit, offset)
}

// RequireActive loads the scope and ensures it is currently usable.
// This is the mandatory gate before any authorization or task work.
//
// Failures:
//   - missing scope → ErrNotFound / ErrScopeInvalid
//   - inactive / revoked / expired status → ErrScopeInvalid
//   - outside time window → ErrScopeInvalid
func (s *Service) RequireActive(ctx context.Context, id contracts.ScopeID, now time.Time) (*scope.Scope, error) {
	sc, err := s.Get(ctx, id)
	if err != nil {
		if err == contracts.ErrNotFound {
			return nil, fmt.Errorf("%w: scope %s not found", contracts.ErrScopeInvalid, id)
		}
		return nil, err
	}
	if err := sc.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", contracts.ErrScopeInvalid, err)
	}
	if !sc.IsActive(now) {
		return nil, fmt.Errorf("%w: scope %s is not active (status=%s)", contracts.ErrScopeInvalid, id, sc.Status)
	}
	return sc, nil
}

// EnsureAllowsAsset checks that the scope permits the given asset.
// An empty allow-list means no restriction at the scope level.
func (s *Service) EnsureAllowsAsset(sc *scope.Scope, assetID contracts.AssetID) error {
	if sc == nil {
		return contracts.ErrScopeInvalid
	}
	if !sc.AllowsAsset(assetID) {
		return fmt.Errorf("%w: asset %s is outside scope %s", contracts.ErrForbidden, assetID, sc.ID)
	}
	return nil
}

// EnsureSameProject verifies the scope belongs to the expected project.
func EnsureSameProject(sc *scope.Scope, projectID contracts.ProjectID) error {
	if sc == nil {
		return contracts.ErrScopeInvalid
	}
	if sc.ProjectID != projectID {
		return fmt.Errorf("%w: scope %s does not belong to project %s", contracts.ErrScopeInvalid, sc.ID, projectID)
	}
	return nil
}
