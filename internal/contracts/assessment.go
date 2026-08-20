package contracts

// AssessmentState is the orchestration lifecycle for an engagement/assessment.
type AssessmentState string

const (
	AssessmentStateCreated   AssessmentState = "CREATED"
	AssessmentStatePlanned   AssessmentState = "PLANNED"
	AssessmentStateRunning   AssessmentState = "RUNNING"
	AssessmentStateSucceeded AssessmentState = "SUCCEEDED"
	AssessmentStateFailed    AssessmentState = "FAILED"
	AssessmentStateCancelled AssessmentState = "CANCELLED"
)

// Valid reports whether the assessment state is known.
func (s AssessmentState) Valid() bool {
	switch s {
	case AssessmentStateCreated, AssessmentStatePlanned, AssessmentStateRunning,
		AssessmentStateSucceeded, AssessmentStateFailed, AssessmentStateCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal reports final assessment states.
func (s AssessmentState) IsTerminal() bool {
	switch s {
	case AssessmentStateSucceeded, AssessmentStateFailed, AssessmentStateCancelled:
		return true
	default:
		return false
	}
}

// AssessmentStep describes one planned tool invocation (capability-based, not tool-hardcoded).
type AssessmentStep struct {
	ID           string       `json:"id"`
	CapabilityID CapabilityID `json:"capability_id"`
	// ExecutorHint optionally pins a registered tool executor id.
	ExecutorHint string         `json:"executor_hint,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
}

// AssessmentID identifies an assessment.
type AssessmentID string
