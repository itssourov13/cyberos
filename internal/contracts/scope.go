package contracts

// ScopeStatus represents the lifecycle state of a scope.
type ScopeStatus string

const (
	ScopeStatusActive   ScopeStatus = "ACTIVE"
	ScopeStatusInactive ScopeStatus = "INACTIVE"
	ScopeStatusExpired  ScopeStatus = "EXPIRED"
	ScopeStatusRevoked  ScopeStatus = "REVOKED"
)

// Valid reports whether the status is known.
func (s ScopeStatus) Valid() bool {
	switch s {
	case ScopeStatusActive, ScopeStatusInactive, ScopeStatusExpired, ScopeStatusRevoked:
		return true
	default:
		return false
	}
}

// ScopeConstraints captures high-level constraints attached to a scope.
// Concrete enforcement lives in later phases; this is the stable shape.
type ScopeConstraints struct {
	// AllowedAssetIDs, if non-empty, restricts the scope to these assets.
	AllowedAssetIDs []AssetID `json:"allowed_asset_ids,omitempty"`

	// AllowedActionClasses are coarse capability classes permitted in this scope.
	AllowedActionClasses []string `json:"allowed_action_classes,omitempty"`

	// Metadata holds opaque scope metadata (never used for authorization decisions
	// without explicit policy evaluation).
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ScopeRef is a lightweight reference to a scope used across contracts.
type ScopeRef struct {
	ID        ScopeID     `json:"id"`
	ProjectID ProjectID   `json:"project_id"`
	Status    ScopeStatus `json:"status"`
	StartsAt  *Timestamp  `json:"starts_at,omitempty"`
	ExpiresAt *Timestamp  `json:"expires_at,omitempty"`
}

// Validate checks required fields and status.
func (s ScopeRef) Validate() error {
	if err := ValidateNonEmpty("scope.id", string(s.ID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("scope.project_id", string(s.ProjectID)); err != nil {
		return err
	}
	if !s.Status.Valid() {
		return ErrInvalidArgument
	}
	return nil
}

// IsActive reports whether the scope is currently usable based on status and time.
// A nil StartsAt/ExpiresAt means no bound on that side.
func (s ScopeRef) IsActive(now Timestamp) bool {
	if s.Status != ScopeStatusActive {
		return false
	}
	if s.StartsAt != nil && now.Before(*s.StartsAt) {
		return false
	}
	if s.ExpiresAt != nil && !now.Before(*s.ExpiresAt) {
		return false
	}
	return true
}
