package contracts

// ExecutionGrantStatus represents the lifecycle of a grant.
type ExecutionGrantStatus string

const (
	ExecutionGrantStatusIssued   ExecutionGrantStatus = "ISSUED"
	ExecutionGrantStatusConsumed ExecutionGrantStatus = "CONSUMED"
	ExecutionGrantStatusExpired  ExecutionGrantStatus = "EXPIRED"
	ExecutionGrantStatusRevoked  ExecutionGrantStatus = "REVOKED"
	ExecutionGrantStatusInvalid  ExecutionGrantStatus = "INVALID"
)

// Valid reports whether the status is known.
func (s ExecutionGrantStatus) Valid() bool {
	switch s {
	case ExecutionGrantStatusIssued, ExecutionGrantStatusConsumed,
		ExecutionGrantStatusExpired, ExecutionGrantStatusRevoked,
		ExecutionGrantStatusInvalid:
		return true
	default:
		return false
	}
}

// ExecutionGrantConstraints limits what the grant may be used for.
// Future phases may add cryptographic binding; Phase 1A only defines the shape.
type ExecutionGrantConstraints struct {
	// MaxUses, if > 0, limits how many times the grant may be consumed.
	MaxUses int `json:"max_uses,omitempty"`

	// AllowedExecutorHints restrict the grant to these registered executors.
	// An empty list delegates to an explicitly configured development default;
	// production issuance must always bind a concrete executor.
	AllowedExecutorHints []string `json:"allowed_executor_hints,omitempty"`

	// AllowedAssetIDs are authoritative asset references approved by the scope.
	AllowedAssetIDs []AssetID `json:"allowed_asset_ids,omitempty"`

	// AllowedTargets bind target-bearing executor parameters (target/url) to
	// canonical asset values or the exact target approved at grant issuance.
	AllowedTargets []string `json:"allowed_targets,omitempty"`

	// Metadata holds opaque, non-authoritative data.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ExecutionGrant is the security bridge between authorization and execution.
//
// Invariant (future enforcement):
//
//	NO VALID GRANT  ⇒  NO EXECUTION
//
// A grant is short-lived, narrowly scoped, task-specific, capability-specific,
// and auditable. Phase 1A only defines the stable contract; issuance and
// cryptographic protection belong to later phases.
type ExecutionGrant struct {
	ID            ExecutionGrantID          `json:"id"`
	Principal     PrincipalRef              `json:"principal"`
	ProjectID     ProjectID                 `json:"project_id"`
	ScopeID       ScopeID                   `json:"scope_id"`
	TaskID        TaskID                    `json:"task_id"`
	CapabilityID  CapabilityID              `json:"capability_id"`
	Status        ExecutionGrantStatus      `json:"status"`
	IssuedAt      Timestamp                 `json:"issued_at"`
	ExpiresAt     Timestamp                 `json:"expires_at"`
	Constraints   ExecutionGrantConstraints `json:"constraints,omitempty"`
	CorrelationID CorrelationID             `json:"correlation_id,omitempty"`
	// Nonce or similar value for future replay protection.
	Nonce string `json:"nonce,omitempty"`

	// PackContentDigest binds the authorized pack content (Phase 19A).
	// Format: sha256:<64 lowercase hex>. Empty = legacy grant without pack binding.
	// Authoritative: set only by GrantIssuer from server-derived trust context.
	PackContentDigest string `json:"pack_content_digest,omitempty"`
	// PackKeyID is signer provenance only — does NOT authorize execution.
	PackKeyID string `json:"pack_key_id,omitempty"`

	// Signature is the issuer's HMAC over the authoritative grant fields.
	// Set only by the server-side GrantIssuer; the Broker rejects grants whose
	// signature does not verify (Phase 24D grant provenance).
	Signature string `json:"signature,omitempty"`
}

// Validate checks structural integrity of the grant.
// It does NOT perform cryptographic verification or database lookups.
func (g ExecutionGrant) Validate() error {
	if err := ValidateNonEmpty("grant.id", string(g.ID)); err != nil {
		return err
	}
	if err := g.Principal.Validate(); err != nil {
		return err
	}
	if err := ValidateNonEmpty("grant.project_id", string(g.ProjectID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("grant.scope_id", string(g.ScopeID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("grant.task_id", string(g.TaskID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("grant.capability_id", string(g.CapabilityID)); err != nil {
		return err
	}
	if !g.Status.Valid() {
		return ErrInvalidArgument
	}
	if g.IssuedAt.IsZero() {
		return ErrInvalidArgument
	}
	if g.ExpiresAt.IsZero() {
		return ErrInvalidArgument
	}
	if !g.ExpiresAt.After(g.IssuedAt) {
		return ErrInvalidArgument
	}
	if g.PackContentDigest != "" {
		if err := validatePackDigestFormat(g.PackContentDigest); err != nil {
			return err
		}
	}
	return nil
}

// validatePackDigestFormat accepts sha256:<64 lowercase hex>.
func validatePackDigestFormat(d string) error {
	const prefix = "sha256:"
	if len(d) != len(prefix)+64 || d[:len(prefix)] != prefix {
		return ErrInvalidArgument
	}
	for _, c := range d[len(prefix):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ErrInvalidArgument
		}
	}
	return nil
}

// IsUsable reports whether the grant is currently usable based on status and time.
// Callers in the Tool Broker must still perform full validation against the
// requesting principal, task, capability, and scope.
func (g ExecutionGrant) IsUsable(now Timestamp) bool {
	if g.Status != ExecutionGrantStatusIssued {
		return false
	}
	if now.Before(g.IssuedAt) {
		return false
	}
	if !now.Before(g.ExpiresAt) {
		return false
	}
	return true
}
