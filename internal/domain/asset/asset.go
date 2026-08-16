// Package asset implements the Asset domain entity.
package asset

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Asset represents a target or resource under management within a project.
type Asset struct {
	ID          contracts.AssetID
	ProjectID   contracts.ProjectID
	Type        contracts.AssetType
	Name        string
	Value       string // e.g. hostname, domain, URL
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// New creates a valid Asset or returns an error.
func New(id contracts.AssetID, projectID contracts.ProjectID, typ contracts.AssetType, name, value string) (*Asset, error) {
	if err := contracts.ValidateNonEmpty("asset.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("asset.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if !typ.Valid() {
		return nil, fmt.Errorf("%w: invalid asset type %q", contracts.ErrInvalidArgument, typ)
	}
	if err := contracts.ValidateNonEmpty("asset.name", name); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("asset.value", value); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Asset{
		ID:        id,
		ProjectID: projectID,
		Type:      typ,
		Name:      name,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (a *Asset) Validate() error {
	if a == nil {
		return fmt.Errorf("%w: asset is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("asset.id", string(a.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("asset.project_id", string(a.ProjectID)); err != nil {
		return err
	}
	if !a.Type.Valid() {
		return fmt.Errorf("%w: invalid asset type %q", contracts.ErrInvalidArgument, a.Type)
	}
	if err := contracts.ValidateNonEmpty("asset.name", a.Name); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("asset.value", a.Value); err != nil {
		return err
	}
	return nil
}

// BelongsTo reports whether the asset belongs to the given project.
func (a *Asset) BelongsTo(projectID contracts.ProjectID) bool {
	return a != nil && a.ProjectID == projectID
}

// Ref returns a lightweight contract reference.
func (a *Asset) Ref() contracts.AssetRef {
	return contracts.AssetRef{ID: a.ID, Type: a.Type}
}
