package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AutonomyOutboxRepo is an in-memory AutonomyOutboxRepository.
type AutonomyOutboxRepo struct {
	mu   sync.Mutex
	data map[string]ports.AutonomyOutboxRecord // id -> record
	byIK map[string]string                     // project|idem -> id
}

func NewAutonomyOutboxRepo() *AutonomyOutboxRepo {
	return &AutonomyOutboxRepo{
		data: make(map[string]ports.AutonomyOutboxRecord),
		byIK: make(map[string]string),
	}
}

func ik(project contracts.ProjectID, key string) string {
	return string(project) + "|" + key
}

func (r *AutonomyOutboxRepo) Enqueue(ctx context.Context, rec ports.AutonomyOutboxRecord) error {
	if rec.ID == "" || rec.ProjectID == "" || rec.IdempotencyKey == "" {
		return fmt.Errorf("%w: id, project_id, idempotency_key required", contracts.ErrInvalidArgument)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	k := ik(rec.ProjectID, rec.IdempotencyKey)
	if existing, ok := r.byIK[k]; ok {
		// idempotent enqueue: already present
		_ = existing
		return nil
	}
	now := time.Now().UTC()
	if rec.Status == "" {
		rec.Status = ports.OutboxPending
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	if rec.AvailableAt.IsZero() {
		rec.AvailableAt = now
	}
	if rec.PayloadJSON == nil {
		rec.PayloadJSON = []byte("{}")
	}
	r.data[rec.ID] = rec
	r.byIK[k] = rec.ID
	return nil
}

func (r *AutonomyOutboxRepo) ClaimBatch(ctx context.Context, workerID string, limit int, lease time.Duration) ([]ports.AutonomyOutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	var candidates []ports.AutonomyOutboxRecord
	for _, rec := range r.data {
		switch rec.Status {
		case ports.OutboxPending:
			if !rec.AvailableAt.After(now) {
				candidates = append(candidates, rec)
			}
		case ports.OutboxClaimed:
			if rec.LockedUntil.IsZero() || rec.LockedUntil.Before(now) {
				candidates = append(candidates, rec)
			}
		}
	}
	// Per-run ordering: sort by autonomy_run_id then created_at
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].AutonomyRunID != candidates[j].AutonomyRunID {
			return candidates[i].AutonomyRunID < candidates[j].AutonomyRunID
		}
		return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
	})
	// One claim per run per batch to preserve sequential processing per run
	seenRun := map[string]bool{}
	var out []ports.AutonomyOutboxRecord
	for _, rec := range candidates {
		if limit > 0 && len(out) >= limit {
			break
		}
		runKey := rec.AutonomyRunID
		if runKey == "" {
			runKey = rec.ID
		}
		if seenRun[runKey] {
			continue
		}
		seenRun[runKey] = true
		rec.Status = ports.OutboxClaimed
		rec.LockedBy = workerID
		rec.LockedUntil = now.Add(lease)
		rec.Attempts++
		rec.UpdatedAt = now
		r.data[rec.ID] = rec
		out = append(out, rec)
	}
	return out, nil
}

func (r *AutonomyOutboxRepo) MarkDelivered(ctx context.Context, id string, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data[id]
	if !ok {
		return contracts.ErrNotFound
	}
	if rec.Status == ports.OutboxClaimed && rec.LockedBy != "" && rec.LockedBy != workerID {
		return fmt.Errorf("%w: worker mismatch", contracts.ErrConflict)
	}
	rec.Status = ports.OutboxDelivered
	rec.LockedBy = ""
	rec.LockedUntil = time.Time{}
	rec.UpdatedAt = time.Now().UTC()
	r.data[id] = rec
	return nil
}

func (r *AutonomyOutboxRepo) MarkRetry(ctx context.Context, id string, workerID string, nextAttempt time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data[id]
	if !ok {
		return contracts.ErrNotFound
	}
	rec.Status = ports.OutboxPending
	rec.AvailableAt = nextAttempt
	rec.LockedBy = ""
	rec.LockedUntil = time.Time{}
	rec.LastError = reason
	rec.UpdatedAt = time.Now().UTC()
	r.data[id] = rec
	return nil
}

func (r *AutonomyOutboxRepo) MarkDeadLetter(ctx context.Context, id string, workerID string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data[id]
	if !ok {
		return contracts.ErrNotFound
	}
	rec.Status = ports.OutboxDeadLetter
	rec.LockedBy = ""
	rec.LockedUntil = time.Time{}
	rec.LastError = reason
	rec.UpdatedAt = time.Now().UTC()
	r.data[id] = rec
	return nil
}

func (r *AutonomyOutboxRepo) Get(ctx context.Context, projectID contracts.ProjectID, id string) (ports.AutonomyOutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.data[id]
	if !ok || rec.ProjectID != projectID {
		return ports.AutonomyOutboxRecord{}, contracts.ErrNotFound
	}
	return rec, nil
}
