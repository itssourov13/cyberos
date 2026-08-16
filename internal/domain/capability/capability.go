// Package capability implements the Capability domain entity.
// Capability discovery is intentionally separate from authorization.
package capability

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Capability describes something that is technically available.
// Presence in the registry does NOT imply any principal is authorized to use it.
type Capability struct {
	ID          contracts.CapabilityID
	Name        string
	Type        contracts.CapabilityType
	Description string
	Version     contracts.SchemaVersion
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// New creates a valid Capability or returns an error.
func New(id contracts.CapabilityID, name string, typ contracts.CapabilityType) (*Capability, error) {
	if err := contracts.ValidateNonEmpty("capability.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("capability.name", name); err != nil {
		return nil, err
	}
	if !typ.Valid() {
		return nil, fmt.Errorf("%w: invalid capability type %q", contracts.ErrInvalidArgument, typ)
	}
	now := contracts.NowUTC()
	return &Capability{
		ID:        id,
		Name:      name,
		Type:      typ,
		Version:   "1",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (c *Capability) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: capability is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("capability.id", string(c.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("capability.name", c.Name); err != nil {
		return err
	}
	if !c.Type.Valid() {
		return fmt.Errorf("%w: invalid capability type %q", contracts.ErrInvalidArgument, c.Type)
	}
	return nil
}

// IsActive reports whether the capability is currently available for discovery.
func (c *Capability) IsActive() bool {
	return c != nil && c.Active
}

// Descriptor returns the contract-level descriptor.
func (c *Capability) Descriptor() contracts.CapabilityDescriptor {
	return contracts.CapabilityDescriptor{
		ID:          c.ID,
		Name:        c.Name,
		Type:        c.Type,
		Description: c.Description,
		Version:     c.Version,
	}
}

// Ref returns a lightweight contract reference.
func (c *Capability) Ref() contracts.CapabilityRef {
	return contracts.CapabilityRef{ID: c.ID}
}
