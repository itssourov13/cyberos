package execution

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// WorkerStatus is the durable lifecycle of a registered execution worker.
type WorkerStatus string

const (
	WorkerPending   WorkerStatus = "PENDING"
	WorkerActive    WorkerStatus = "ACTIVE"
	WorkerSuspended WorkerStatus = "SUSPENDED"
	WorkerRevoked   WorkerStatus = "REVOKED"
	WorkerExpired   WorkerStatus = "EXPIRED"
)

// Worker is an authenticated remote execution endpoint identity (Phase 17C).
// It is NOT an authorization authority and does not execute in 17C.
type Worker struct {
	ID              string
	ProjectID       contracts.ProjectID
	TenantID        contracts.TenantID
	HostID          HostID // logical host id bound to this worker
	Status          WorkerStatus
	CredentialRef   string // secret reference only — never plaintext
	CertFingerprint string // public identity (e.g. SHA256 hex)
	Capabilities    map[string]string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSeenAt      time.Time
	ExpiresAt       time.Time
	Version         int64
}

// CanAuthenticate reports whether the worker identity may present credentials.
// Does NOT authorize execution.
func (w Worker) CanAuthenticate(now time.Time) error {
	if w.ID == "" {
		return fmt.Errorf("%w: unknown worker", contracts.ErrNotFound)
	}
	switch w.Status {
	case WorkerRevoked:
		return fmt.Errorf("%w: worker revoked", contracts.ErrForbidden)
	case WorkerExpired:
		return fmt.Errorf("%w: worker expired", contracts.ErrForbidden)
	case WorkerSuspended:
		return fmt.Errorf("%w: worker suspended", contracts.ErrForbidden)
	case WorkerPending, WorkerActive:
		// ok
	default:
		return fmt.Errorf("%w: invalid worker status", contracts.ErrInvalidArgument)
	}
	if !w.ExpiresAt.IsZero() && !now.Before(w.ExpiresAt) {
		return fmt.Errorf("%w: worker registration expired", contracts.ErrForbidden)
	}
	return nil
}

// ValidTransition enforces lifecycle rules (fail closed).
func ValidTransition(from, to WorkerStatus) bool {
	switch from {
	case WorkerPending:
		return to == WorkerActive || to == WorkerRevoked || to == WorkerExpired
	case WorkerActive:
		return to == WorkerSuspended || to == WorkerRevoked || to == WorkerExpired
	case WorkerSuspended:
		return to == WorkerActive || to == WorkerRevoked || to == WorkerExpired
	case WorkerRevoked, WorkerExpired:
		return false // terminal — no resurrection
	default:
		return false
	}
}
