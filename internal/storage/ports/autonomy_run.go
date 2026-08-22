package ports

import (
	"context"
	"time"

	"cyberos/internal/contracts"
)

// AutonomyRunRecord is durable autonomy state (Phase 14A). No Grant/secrets.
type AutonomyRunRecord struct {
	ID            string
	ProjectID     contracts.ProjectID
	TenantID      contracts.TenantID
	ScopeID       contracts.ScopeID
	PrincipalID   contracts.PrincipalID
	CorrelationID contracts.CorrelationID
	Goal          string
	Mode          string
	PolicyVersion string
	PolicyJSON    []byte // snapshot
	State         string
	Iteration     int
	StepsConsumed int
	AnalysisID    string
	PlanID        string
	ParentPlanID  string
	WorkflowDefID string
	WorkflowRunID string
	LastDecision  string
	StopReason    string
	Version       int64 // optimistic concurrency
	StartedAt     time.Time
	UpdatedAt     time.Time
	FinishedAt    time.Time
	CancelledAt   time.Time
}

// AutonomyRunRepository persists AutonomyRun state for restart/resume.
type AutonomyRunRepository interface {
	Create(ctx context.Context, r AutonomyRunRecord) error
	Get(ctx context.Context, projectID contracts.ProjectID, runID string) (AutonomyRunRecord, error)
	Update(ctx context.Context, r AutonomyRunRecord) error
	Cancel(ctx context.Context, projectID contracts.ProjectID, runID string) error
	ListResumable(ctx context.Context, projectID contracts.ProjectID) ([]AutonomyRunRecord, error)
}
