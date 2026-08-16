// Package principal implements the Principal domain entity.
package principal

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Principal is a first-class actor in CYBEROS.
// Supported types begin with HUMAN; the model is extensible for SERVICE, AGENT, etc.
type Principal struct {
	ID          contracts.PrincipalID
	Type        contracts.PrincipalType
	Name        string
	DisplayName string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// New creates a valid Principal or returns an error.
func New(id contracts.PrincipalID, typ contracts.PrincipalType, name string) (*Principal, error) {
	if err := contracts.ValidateNonEmpty("principal.id", string(id)); err != nil {
		return nil, err
	}
	if !typ.Valid() {
		return nil, fmt.Errorf("%w: invalid principal type %q", contracts.ErrInvalidArgument, typ)
	}
	if err := contracts.ValidateNonEmpty("principal.name", name); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Principal{
		ID:        id,
		Type:      typ,
		Name:      name,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]string),
	}, nil
}

// NewHuman is a convenience constructor for human principals.
func NewHuman(id contracts.PrincipalID, name string) (*Principal, error) {
	return New(id, contracts.PrincipalTypeHuman, name)
}

// Validate checks structural invariants.
func (p *Principal) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: principal is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("principal.id", string(p.ID)); err != nil {
		return err
	}
	if !p.Type.Valid() {
		return fmt.Errorf("%w: invalid principal type %q", contracts.ErrInvalidArgument, p.Type)
	}
	if err := contracts.ValidateNonEmpty("principal.name", p.Name); err != nil {
		return err
	}
	return nil
}

// IsActive reports whether the principal may initiate work.
func (p *Principal) IsActive() bool {
	return p != nil && p.Active
}

// Ref returns a lightweight contract reference.
func (p *Principal) Ref() contracts.PrincipalRef {
	return contracts.PrincipalRef{ID: p.ID, Type: p.Type}
}
