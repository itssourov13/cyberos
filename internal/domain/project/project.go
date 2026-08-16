// Package project implements the Project domain entity.
package project

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Status represents the lifecycle of a project.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusArchived Status = "ARCHIVED"
	StatusDeleted  Status = "DELETED"
)

// Valid reports whether the status is known.
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusArchived, StatusDeleted:
		return true
	default:
		return false
	}
}

// Project is the top-level container for assets, scopes, and tasks.
type Project struct {
	ID          contracts.ProjectID
	Name        string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// New creates a valid Project or returns an error.
func New(id contracts.ProjectID, name string) (*Project, error) {
	if err := contracts.ValidateNonEmpty("project.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("project.name", name); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Project{
		ID:        id,
		Name:      name,
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (p *Project) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: project is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("project.id", string(p.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("project.name", p.Name); err != nil {
		return err
	}
	if !p.Status.Valid() {
		return fmt.Errorf("%w: invalid project status %q", contracts.ErrInvalidArgument, p.Status)
	}
	if p.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at is required", contracts.ErrInvalidArgument)
	}
	return nil
}

// IsActive reports whether the project can accept new work.
func (p *Project) IsActive() bool {
	return p != nil && p.Status == StatusActive
}

// Ref returns a lightweight contract reference.
func (p *Project) Ref() contracts.ProjectRef {
	return contracts.ProjectRef{ID: p.ID}
}
