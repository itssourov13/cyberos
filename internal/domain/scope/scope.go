// Package scope implements the Scope domain entity — the root authorization boundary.
package scope

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Scope is the root authorization boundary for work in CYBEROS.
// A task cannot receive authorization outside its scope.
// Expired or inactive scopes must fail authorization (enforced by later layers).
type Scope struct {
	ID          contracts.ScopeID
	ProjectID   contracts.ProjectID
	Name        string
	Status      contracts.ScopeStatus
	StartsAt    *time.Time
	ExpiresAt   *time.Time
	Constraints contracts.ScopeConstraints
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// New creates a valid Scope or returns an error.
func New(id contracts.ScopeID, projectID contracts.ProjectID, name string, status contracts.ScopeStatus) (*Scope, error) {
	if err := contracts.ValidateNonEmpty("scope.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("scope.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("scope.name", name); err != nil {
		return nil, err
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: invalid scope status %q", contracts.ErrInvalidArgument, status)
	}
	now := contracts.NowUTC()
	return &Scope{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (s *Scope) Validate() error {
	if s == nil {
		return fmt.Errorf("%w: scope is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("scope.id", string(s.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("scope.project_id", string(s.ProjectID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("scope.name", s.Name); err != nil {
		return err
	}
	if !s.Status.Valid() {
		return fmt.Errorf("%w: invalid scope status %q", contracts.ErrInvalidArgument, s.Status)
	}
	if s.StartsAt != nil && s.ExpiresAt != nil && !s.ExpiresAt.After(*s.StartsAt) {
		return fmt.Errorf("%w: expires_at must be after starts_at", contracts.ErrInvalidArgument)
	}
	return nil
}

// IsActive reports whether the scope is currently usable.
// Mirrors the contract helper but operates on the full domain entity.
func (s *Scope) IsActive(now time.Time) bool {
	if s == nil {
		return false
	}
	ref := s.Ref()
	return ref.IsActive(now)
}

// AllowsAsset reports whether the given asset is permitted by the scope constraints.
// An empty AllowedAssetIDs list means no restriction at this level.
func (s *Scope) AllowsAsset(assetID contracts.AssetID) bool {
	if s == nil {
		return false
	}
	if len(s.Constraints.AllowedAssetIDs) == 0 {
		return true
	}
	for _, id := range s.Constraints.AllowedAssetIDs {
		if id == assetID {
			return true
		}
	}
	return false
}

// Ref returns a lightweight contract reference.
func (s *Scope) Ref() contracts.ScopeRef {
	var starts, expires *contracts.Timestamp
	if s.StartsAt != nil {
		t := *s.StartsAt
		starts = &t
	}
	if s.ExpiresAt != nil {
		t := *s.ExpiresAt
		expires = &t
	}
	return contracts.ScopeRef{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Status:    s.Status,
		StartsAt:  starts,
		ExpiresAt: expires,
	}
}
