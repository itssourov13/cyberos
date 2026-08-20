package assessment

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Assessment is an engagement bound to project, scope, and principal.
// Orchestration never replaces Control Plane authorization.
type Assessment struct {
	ID          contracts.AssessmentID
	ProjectID   contracts.ProjectID
	ScopeID     contracts.ScopeID
	PrincipalID contracts.PrincipalID
	Title       string
	Target      string // asset identifier / hostname label (must fit scope constraints upstream)
	State       contracts.AssessmentState
	Steps       []contracts.AssessmentStep
	// StepResults parallel Steps after execution attempts.
	StepResults []StepResult
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Cancelled   bool
}

// StepResult records one step outcome without embedding secrets.
type StepResult struct {
	StepID       string
	CapabilityID contracts.CapabilityID
	Success      bool
	Summary      string
	GrantID      contracts.ExecutionGrantID
	ExecutionID  string
	EvidenceID   contracts.EvidenceID
	ErrorMessage string
	CompletedAt  time.Time
}

// New creates an assessment in CREATED state.
func New(
	id contracts.AssessmentID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	principalID contracts.PrincipalID,
	title, target string,
) (*Assessment, error) {
	if err := contracts.ValidateNonEmpty("assessment.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("assessment.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("assessment.scope_id", string(scopeID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("assessment.principal_id", string(principalID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("assessment.title", title); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Assessment{
		ID: id, ProjectID: projectID, ScopeID: scopeID, PrincipalID: principalID,
		Title: title, Target: target, State: contracts.AssessmentStateCreated,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

var allowed = map[contracts.AssessmentState][]contracts.AssessmentState{
	contracts.AssessmentStateCreated: {
		contracts.AssessmentStatePlanned,
		contracts.AssessmentStateCancelled,
	},
	contracts.AssessmentStatePlanned: {
		contracts.AssessmentStateRunning,
		contracts.AssessmentStateCancelled,
	},
	contracts.AssessmentStateRunning: {
		contracts.AssessmentStateSucceeded,
		contracts.AssessmentStateFailed,
		contracts.AssessmentStateCancelled,
	},
}

// TransitionTo validates and applies a state change.
func (a *Assessment) TransitionTo(next contracts.AssessmentState) error {
	if a == nil {
		return contracts.ErrInvalidArgument
	}
	if a.Cancelled && next != contracts.AssessmentStateCancelled {
		return fmt.Errorf("%w: assessment cancelled", contracts.ErrInvalidStateTransition)
	}
	ok := false
	for _, s := range allowed[a.State] {
		if s == next {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("%w: %s → %s", contracts.ErrInvalidStateTransition, a.State, next)
	}
	a.State = next
	a.UpdatedAt = contracts.NowUTC()
	if next == contracts.AssessmentStateCancelled {
		a.Cancelled = true
	}
	return nil
}

// Plan attaches sequential steps and moves to PLANNED.
func (a *Assessment) Plan(steps []contracts.AssessmentStep) error {
	if a == nil {
		return contracts.ErrInvalidArgument
	}
	if a.State != contracts.AssessmentStateCreated {
		return fmt.Errorf("%w: plan only from CREATED", contracts.ErrInvalidStateTransition)
	}
	for i, s := range steps {
		if s.ID == "" || s.CapabilityID == "" {
			return fmt.Errorf("%w: step %d missing id/capability", contracts.ErrInvalidArgument, i)
		}
	}
	a.Steps = append([]contracts.AssessmentStep{}, steps...)
	return a.TransitionTo(contracts.AssessmentStatePlanned)
}

// Cancel requests cancellation.
func (a *Assessment) Cancel() error {
	if a == nil {
		return contracts.ErrInvalidArgument
	}
	if a.State.IsTerminal() {
		return fmt.Errorf("%w: already terminal", contracts.ErrInvalidStateTransition)
	}
	return a.TransitionTo(contracts.AssessmentStateCancelled)
}
