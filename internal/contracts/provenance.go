package contracts

// Provenance answers the forensic questions for evidence and findings:
//
//	WHO?   WHAT?   WHEN?   WHERE?   WHY?
//	UNDER WHICH SCOPE / TASK / CAPABILITY?
//	PRODUCED BY WHICH TOOL / AGENT / EXECUTOR?
//
// Phase 1A defines only the stable shape. Full provenance construction and
// cryptographic chaining belong to later phases.
type Provenance struct {
	// Subject identifies what this provenance is about (evidence or finding).
	SubjectType string `json:"subject_type"` // "evidence" | "finding"
	SubjectID   string `json:"subject_id"`

	// Actor
	Principal PrincipalRef `json:"principal"`

	// Context
	ProjectID    ProjectID        `json:"project_id"`
	ScopeID      ScopeID          `json:"scope_id"`
	TaskID       TaskID           `json:"task_id"`
	CapabilityID CapabilityID     `json:"capability_id"`
	GrantID      ExecutionGrantID `json:"grant_id,omitempty"`

	// Execution
	ExecutorID string    `json:"executor_id,omitempty"`
	ProducedAt Timestamp `json:"produced_at"`

	// Optional lineage
	ParentIDs     []string      `json:"parent_ids,omitempty"`
	CorrelationID CorrelationID `json:"correlation_id,omitempty"`

	// Integrity
	Digest string `json:"digest,omitempty"`
}

// Validate checks required provenance fields.
func (p Provenance) Validate() error {
	if err := ValidateNonEmpty("provenance.subject_type", p.SubjectType); err != nil {
		return err
	}
	if err := ValidateNonEmpty("provenance.subject_id", p.SubjectID); err != nil {
		return err
	}
	if err := p.Principal.Validate(); err != nil {
		return err
	}
	if err := ValidateNonEmpty("provenance.project_id", string(p.ProjectID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("provenance.scope_id", string(p.ScopeID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("provenance.task_id", string(p.TaskID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("provenance.capability_id", string(p.CapabilityID)); err != nil {
		return err
	}
	if p.ProducedAt.IsZero() {
		return ErrInvalidArgument
	}
	return nil
}
