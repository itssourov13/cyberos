package tenant

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Tenant is an organization boundary for multi-tenant isolation.
// Isolation is not authorization: Control Plane + Grant remain mandatory for execution.
type Tenant struct {
	ID        contracts.TenantID
	Name      string
	Status    contracts.TenantStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]string
}

// New creates an active tenant.
func New(id contracts.TenantID, name string) (*Tenant, error) {
	if err := contracts.ValidateNonEmpty("tenant.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("tenant.name", name); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Tenant{
		ID: id, Name: name, Status: contracts.TenantStatusActive,
		CreatedAt: now, UpdatedAt: now, Metadata: map[string]string{},
	}, nil
}

// IsActive reports whether the tenant may own resources.
func (t *Tenant) IsActive() bool {
	return t != nil && t.Status == contracts.TenantStatusActive
}

// Validate checks required fields.
func (t *Tenant) Validate() error {
	if t == nil {
		return fmt.Errorf("%w: tenant nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("tenant.id", string(t.ID)); err != nil {
		return err
	}
	if !t.Status.Valid() {
		return contracts.ErrInvalidArgument
	}
	return nil
}
