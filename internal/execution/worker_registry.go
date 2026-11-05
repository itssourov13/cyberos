package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mtls"
	"cyberos/internal/storage/ports"
)

// WorkerRegistry manages worker identity lifecycle. It never executes work
// and never mints Grants or calls CP.
type WorkerRegistry struct {
	mu   sync.Mutex
	repo ports.WorkerRepository
	now  func() time.Time
}

// NewWorkerRegistry constructs a registry over a durable or memory repository.
func NewWorkerRegistry(repo ports.WorkerRepository) *WorkerRegistry {
	return &WorkerRegistry{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// Register creates a PENDING or ACTIVE worker under a project. HostID must be logical.
func (r *WorkerRegistry) Register(ctx context.Context, w Worker) (Worker, error) {
	if r == nil || r.repo == nil {
		return Worker{}, fmt.Errorf("%w: worker registry not configured", contracts.ErrInvalidArgument)
	}
	if w.ID == "" || w.ProjectID == "" {
		return Worker{}, fmt.Errorf("%w: worker_id and project_id required", contracts.ErrInvalidArgument)
	}
	if err := ValidateHostID(w.HostID); err != nil {
		return Worker{}, err
	}
	if w.CredentialRef == "" && w.CertFingerprint == "" {
		return Worker{}, fmt.Errorf("%w: credential_ref or cert_fingerprint required", contracts.ErrInvalidArgument)
	}
	now := r.now()
	if w.Status == "" {
		w.Status = WorkerPending
	}
	if w.Status != WorkerPending && w.Status != WorkerActive {
		return Worker{}, fmt.Errorf("%w: initial status must be PENDING or ACTIVE", contracts.ErrInvalidArgument)
	}
	w.CreatedAt = now
	w.UpdatedAt = now
	w.Version = 1
	rec := workerToRecord(w)
	if err := r.repo.Register(ctx, rec); err != nil {
		return Worker{}, err
	}
	return w, nil
}

// Get returns a project-scoped worker.
func (r *WorkerRegistry) Get(ctx context.Context, projectID contracts.ProjectID, workerID string) (Worker, error) {
	rec, err := r.repo.Get(ctx, projectID, workerID)
	if err != nil {
		return Worker{}, err
	}
	return recordToWorker(rec), nil
}

// List returns every worker registered under the project (durable repository order).
func (r *WorkerRegistry) List(ctx context.Context, projectID contracts.ProjectID) ([]Worker, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("%w: worker registry not configured", contracts.ErrInvalidArgument)
	}
	recs, err := r.repo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]Worker, 0, len(recs))
	for _, rec := range recs {
		out = append(out, recordToWorker(rec))
	}
	return out, nil
}

// Activate transitions PENDING/SUSPENDED → ACTIVE.
func (r *WorkerRegistry) Activate(ctx context.Context, projectID contracts.ProjectID, workerID string) (Worker, error) {
	return r.transition(ctx, projectID, workerID, WorkerActive)
}

// Suspend transitions ACTIVE → SUSPENDED.
func (r *WorkerRegistry) Suspend(ctx context.Context, projectID contracts.ProjectID, workerID string) (Worker, error) {
	return r.transition(ctx, projectID, workerID, WorkerSuspended)
}

// Revoke is terminal.
func (r *WorkerRegistry) Revoke(ctx context.Context, projectID contracts.ProjectID, workerID string) (Worker, error) {
	return r.transition(ctx, projectID, workerID, WorkerRevoked)
}

func (r *WorkerRegistry) transition(ctx context.Context, projectID contracts.ProjectID, workerID string, to WorkerStatus) (Worker, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return Worker{}, err
	}
	if !ValidTransition(w.Status, to) {
		return Worker{}, fmt.Errorf("%w: cannot transition %s → %s", contracts.ErrConflict, w.Status, to)
	}
	w.Status = to
	w.UpdatedAt = r.now()
	if err := r.repo.UpdateStatus(ctx, projectID, workerID, string(to), w.Version); err != nil {
		return Worker{}, err
	}
	w.Version++
	return w, nil
}

// Heartbeat updates last_seen_at. Does NOT authorize execution.
func (r *WorkerRegistry) Heartbeat(ctx context.Context, projectID contracts.ProjectID, workerID string) (Worker, error) {
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return Worker{}, err
	}
	if w.Status != WorkerActive {
		return Worker{}, fmt.Errorf("%w: worker is not active", contracts.ErrForbidden)
	}
	if err := w.CanAuthenticate(r.now()); err != nil {
		return Worker{}, err
	}
	at := r.now()
	if err := r.repo.TouchHeartbeat(ctx, projectID, workerID, at); err != nil {
		return Worker{}, err
	}
	w.LastSeenAt = at
	w.UpdatedAt = at
	return w, nil
}

// Authenticate verifies identity metadata (fingerprint) for an ACTIVE/PENDING worker.
// Does not grant execution rights.
func (r *WorkerRegistry) Authenticate(ctx context.Context, projectID contracts.ProjectID, workerID, certFingerprint string) (Worker, error) {
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return Worker{}, err
	}
	if err := w.CanAuthenticate(r.now()); err != nil {
		return Worker{}, err
	}
	if w.CertFingerprint != "" && certFingerprint != "" && w.CertFingerprint != certFingerprint {
		return Worker{}, fmt.Errorf("%w: certificate fingerprint mismatch", contracts.ErrForbidden)
	}
	if w.CertFingerprint != "" && certFingerprint == "" {
		return Worker{}, fmt.Errorf("%w: certificate required", contracts.ErrForbidden)
	}
	return w, nil
}

func workerToRecord(w Worker) ports.WorkerRecord {
	caps := w.Capabilities
	if caps == nil {
		caps = map[string]string{}
	}
	return ports.WorkerRecord{
		ID: w.ID, ProjectID: w.ProjectID, TenantID: w.TenantID, HostID: string(w.HostID),
		Status: string(w.Status), CredentialRef: w.CredentialRef, CertFingerprint: w.CertFingerprint,
		Capabilities: caps, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt, LastSeenAt: w.LastSeenAt,
		ExpiresAt: w.ExpiresAt, Version: w.Version,
	}
}

func recordToWorker(rec ports.WorkerRecord) Worker {
	return Worker{
		ID: rec.ID, ProjectID: rec.ProjectID, TenantID: rec.TenantID, HostID: HostID(rec.HostID),
		Status: WorkerStatus(rec.Status), CredentialRef: rec.CredentialRef, CertFingerprint: rec.CertFingerprint,
		Capabilities: rec.Capabilities, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
		LastSeenAt: rec.LastSeenAt, ExpiresAt: rec.ExpiresAt, Version: rec.Version,
	}
}

// AuthenticateTLS binds a verified TLS peer identity to a registered worker (Phase 22A).
// peerFingerprint must come from mtls.ExtractPeerIdentity after a successful handshake.
// Does NOT authorize execution or mint Grants.
func (r *WorkerRegistry) AuthenticateTLS(ctx context.Context, projectID contracts.ProjectID, workerID, hostID, peerFingerprint string) (Worker, error) {
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return Worker{}, err
	}
	if string(w.HostID) != hostID {
		return Worker{}, fmt.Errorf("%w: worker host mismatch", contracts.ErrForbidden)
	}
	if err := mtls.Bind(mtls.PeerIdentity{Fingerprint: peerFingerprint}, mtls.BindRequest{
		ProjectID:   string(projectID),
		WorkerID:    workerID,
		HostID:      hostID,
		Fingerprint: w.CertFingerprint,
		Status:      string(w.Status),
	}); err != nil {
		return Worker{}, fmt.Errorf("%w: %v", contracts.ErrForbidden, err)
	}
	if w.Status != WorkerActive {
		return Worker{}, fmt.Errorf("%w: worker not active", contracts.ErrForbidden)
	}
	if err := w.CanAuthenticate(r.now()); err != nil {
		return Worker{}, err
	}
	return w, nil
}
