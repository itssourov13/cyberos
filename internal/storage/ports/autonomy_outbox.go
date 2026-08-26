package ports

import (
	"context"
	"time"

	"cyberos/internal/contracts"
)

// Outbox status values (Phase 15C).
const (
	OutboxPending    = "PENDING"
	OutboxClaimed    = "CLAIMED"
	OutboxDelivered  = "DELIVERED"
	OutboxDeadLetter = "DEAD_LETTER"
)

// AutonomyOutboxRecord is a durable autonomy event delivery record.
// No Grant, secrets, or authorization tokens.
type AutonomyOutboxRecord struct {
	ID             string
	ProjectID      contracts.ProjectID
	TenantID       contracts.TenantID
	ScopeID        contracts.ScopeID
	AutonomyRunID  string
	WorkflowRunID  string
	ExecutionID    string
	EventType      string
	SchemaVersion  string
	IdempotencyKey string
	EventID        string
	PayloadJSON    []byte
	Status         string
	Attempts       int
	AvailableAt    time.Time
	LockedUntil    time.Time
	LockedBy       string
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AutonomyOutboxRepository persists and claims autonomy domain events for dispatch.
type AutonomyOutboxRepository interface {
	Enqueue(ctx context.Context, rec AutonomyOutboxRecord) error
	ClaimBatch(ctx context.Context, workerID string, limit int, lease time.Duration) ([]AutonomyOutboxRecord, error)
	MarkDelivered(ctx context.Context, id string, workerID string) error
	MarkRetry(ctx context.Context, id string, workerID string, nextAttempt time.Time, reason string) error
	MarkDeadLetter(ctx context.Context, id string, workerID string, reason string) error
	Get(ctx context.Context, projectID contracts.ProjectID, id string) (AutonomyOutboxRecord, error)
}
