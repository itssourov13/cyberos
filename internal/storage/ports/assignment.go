package ports

import (
	"context"
	"time"

	"cyberos/internal/contracts"
)

// AssignmentRecord is durable remote execution assignment state (Phase 17D).
// No Grant, secrets, or private keys.
type AssignmentRecord struct {
	ID            string
	ExecutionID   string
	WorkflowRunID string
	ProjectID     contracts.ProjectID
	TenantID      contracts.TenantID
	HostID        string
	WorkerID      string
	Status        string
	Attempt       int
	MaxAttempts   int
	LeaseUntil    time.Time
	CreatedAt     time.Time
	DispatchedAt  time.Time
	StartedAt     time.Time
	CompletedAt   time.Time
	CancelledAt   time.Time
	ResultHash    string
	ResultStatus  string
	ErrorMessage  string
	Version       int64
}

// AssignmentRepository persists remote execution assignments.
type AssignmentRepository interface {
	Create(ctx context.Context, a AssignmentRecord) error
	Get(ctx context.Context, projectID contracts.ProjectID, id string) (AssignmentRecord, error)
	GetByExecutionID(ctx context.Context, projectID contracts.ProjectID, executionID string) (AssignmentRecord, error)
	Update(ctx context.Context, a AssignmentRecord) error
}
