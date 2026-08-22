package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AutonomyRunRepo is an in-memory AutonomyRunRepository.
type AutonomyRunRepo struct {
	mu   sync.RWMutex
	data map[string]ports.AutonomyRunRecord // key: project|id
}

func NewAutonomyRunRepo() *AutonomyRunRepo {
	return &AutonomyRunRepo{data: make(map[string]ports.AutonomyRunRecord)}
}

func key(project contracts.ProjectID, id string) string {
	return string(project) + "|" + id
}

func (r *AutonomyRunRepo) Create(ctx context.Context, rec ports.AutonomyRunRecord) error {
	if rec.ID == "" || rec.ProjectID == "" {
		return fmt.Errorf("%w: id and project_id required", contracts.ErrInvalidArgument)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(rec.ProjectID, rec.ID)
	if _, ok := r.data[k]; ok {
		return fmt.Errorf("%w: autonomy run exists", contracts.ErrConflict)
	}
	if rec.Version == 0 {
		rec.Version = 1
	}
	if rec.StartedAt.IsZero() {
		rec.StartedAt = time.Now().UTC()
	}
	rec.UpdatedAt = time.Now().UTC()
	r.data[k] = rec
	return nil
}

func (r *AutonomyRunRepo) Get(ctx context.Context, projectID contracts.ProjectID, runID string) (ports.AutonomyRunRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[key(projectID, runID)]
	if !ok {
		return ports.AutonomyRunRecord{}, contracts.ErrNotFound
	}
	return rec, nil
}

func (r *AutonomyRunRepo) Update(ctx context.Context, rec ports.AutonomyRunRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(rec.ProjectID, rec.ID)
	cur, ok := r.data[k]
	if !ok {
		return contracts.ErrNotFound
	}
	if rec.Version != 0 && cur.Version != rec.Version {
		return fmt.Errorf("%w: version conflict", contracts.ErrConflict)
	}
	rec.Version = cur.Version + 1
	rec.UpdatedAt = time.Now().UTC()
	r.data[k] = rec
	return nil
}

func (r *AutonomyRunRepo) Cancel(ctx context.Context, projectID contracts.ProjectID, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(projectID, runID)
	cur, ok := r.data[k]
	if !ok {
		return contracts.ErrNotFound
	}
	if cur.State == "CANCELLED" {
		return nil
	}
	now := time.Now().UTC()
	cur.State = "CANCELLED"
	cur.StopReason = "operator_cancelled"
	cur.CancelledAt = now
	cur.FinishedAt = now
	cur.UpdatedAt = now
	cur.Version++
	r.data[k] = cur
	return nil
}

func (r *AutonomyRunRepo) ListResumable(ctx context.Context, projectID contracts.ProjectID) ([]ports.AutonomyRunRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ports.AutonomyRunRecord
	prefix := string(projectID) + "|"
	for k, rec := range r.data {
		if len(k) < len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		switch rec.State {
		case "RUNNING", "WAITING", "PENDING":
			out = append(out, rec)
		}
	}
	return out, nil
}
