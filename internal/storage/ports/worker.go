package ports

import (
	"context"
	"time"

	"cyberos/internal/contracts"
)

// WorkerRecord is durable worker identity (Phase 17C). No private keys or Grants.
type WorkerRecord struct {
	ID              string
	ProjectID       contracts.ProjectID
	TenantID        contracts.TenantID
	HostID          string
	Status          string
	CredentialRef   string
	CertFingerprint string
	Capabilities    map[string]string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSeenAt      time.Time
	ExpiresAt       time.Time
	Version         int64
}

// WorkerRepository persists worker identity for registration/auth (not execution).
type WorkerRepository interface {
	Register(ctx context.Context, w WorkerRecord) error
	Get(ctx context.Context, projectID contracts.ProjectID, workerID string) (WorkerRecord, error)
	List(ctx context.Context, projectID contracts.ProjectID) ([]WorkerRecord, error)
	UpdateStatus(ctx context.Context, projectID contracts.ProjectID, workerID, status string, version int64) error
	TouchHeartbeat(ctx context.Context, projectID contracts.ProjectID, workerID string, at time.Time) error
	Revoke(ctx context.Context, projectID contracts.ProjectID, workerID string) error
}
