package contracts

// DecisionOutcome is the top-level result of a Control Plane evaluation.
type DecisionOutcome string

const (
	DecisionAllowed DecisionOutcome = "ALLOWED"
	DecisionDenied  DecisionOutcome = "DENIED"
)

// Valid reports whether the outcome is known.
func (o DecisionOutcome) Valid() bool {
	return o == DecisionAllowed || o == DecisionDenied
}

// Decision is the structured result of a Control Plane evaluation.
// It is never a bare boolean. Every sensitive operation must be able
// to produce and record a Decision of this shape.
type Decision struct {
	ID        DecisionID      `json:"id"`
	Outcome   DecisionOutcome `json:"outcome"`
	Reason    string          `json:"reason"`
	Timestamp Timestamp       `json:"timestamp"`

	// Context of the request
	Principal    PrincipalRef `json:"principal"`
	ProjectID    ProjectID    `json:"project_id,omitempty"`
	ScopeID      ScopeID      `json:"scope_id,omitempty"`
	TaskID       TaskID       `json:"task_id,omitempty"`
	CapabilityID CapabilityID `json:"capability_id,omitempty"`

	// Evaluation results
	Policy     *PolicyResult     `json:"policy,omitempty"`
	Permission *PermissionResult `json:"permission,omitempty"`
	Risk       *RiskResult       `json:"risk,omitempty"`
	Approval   *ApprovalResult   `json:"approval,omitempty"`

	// Correlation for tracing and audit
	CorrelationID CorrelationID `json:"correlation_id,omitempty"`

	// References to policies, rules, or other artifacts that influenced the decision.
	References []string `json:"references,omitempty"`
}

// Validate checks structural integrity of the decision.
func (d Decision) Validate() error {
	if err := ValidateNonEmpty("decision.id", string(d.ID)); err != nil {
		return err
	}
	if !d.Outcome.Valid() {
		return ErrInvalidArgument
	}
	if err := ValidateNonEmpty("decision.reason", d.Reason); err != nil {
		return err
	}
	if d.Timestamp.IsZero() {
		return ErrInvalidArgument
	}
	if err := d.Principal.Validate(); err != nil {
		return err
	}
	return nil
}

// Allowed is a convenience helper.
func (d Decision) Allowed() bool {
	return d.Outcome == DecisionAllowed
}
