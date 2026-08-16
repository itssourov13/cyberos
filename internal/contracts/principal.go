package contracts

// PrincipalType classifies who or what is acting.
type PrincipalType string

const (
	PrincipalTypeHuman          PrincipalType = "HUMAN"
	PrincipalTypeService        PrincipalType = "SERVICE"
	PrincipalTypeAgent          PrincipalType = "AGENT"
	PrincipalTypePlugin         PrincipalType = "PLUGIN"
	PrincipalTypeZOYA           PrincipalType = "ZOYA"
	PrincipalTypeExternalClient PrincipalType = "EXTERNAL_CLIENT"
	PrincipalTypeAutomation     PrincipalType = "AUTOMATION"
)

// Valid reports whether the principal type is one of the known values.
func (t PrincipalType) Valid() bool {
	switch t {
	case PrincipalTypeHuman,
		PrincipalTypeService,
		PrincipalTypeAgent,
		PrincipalTypePlugin,
		PrincipalTypeZOYA,
		PrincipalTypeExternalClient,
		PrincipalTypeAutomation:
		return true
	default:
		return false
	}
}

// PrincipalRef is a lightweight reference to a principal used in contracts.
// Full principal records live in the domain / identity layers.
type PrincipalRef struct {
	ID   PrincipalID   `json:"id"`
	Type PrincipalType `json:"type"`
}

// Validate checks required fields.
func (p PrincipalRef) Validate() error {
	if err := ValidateNonEmpty("principal.id", string(p.ID)); err != nil {
		return err
	}
	if !p.Type.Valid() {
		return ErrInvalidArgument
	}
	return nil
}
