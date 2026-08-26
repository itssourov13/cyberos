package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// WorkerRepo is an in-memory WorkerRepository.
type WorkerRepo struct {
	mu      sync.RWMutex
	byKey   map[string]ports.WorkerRecord // project|worker
	hostKey map[string]string             // project|host → worker id
}

func NewWorkerRepo() *WorkerRepo {
	return &WorkerRepo{byKey: make(map[string]ports.WorkerRecord), hostKey: make(map[string]string)}
}

func workerKey(p contracts.ProjectID, id string) string { return string(p) + "|" + id }

func (r *WorkerRepo) Register(_ context.Context, w ports.WorkerRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := workerKey(w.ProjectID, w.ID)
	if _, ok := r.byKey[k]; ok {
		return contracts.ErrConflict
	}
	hk := workerKey(w.ProjectID, w.HostID)
	if _, ok := r.hostKey[hk]; ok {
		return fmt.Errorf("%w: host_id already bound", contracts.ErrConflict)
	}
	r.byKey[k] = w
	r.hostKey[hk] = w.ID
	return nil
}

func (r *WorkerRepo) Get(_ context.Context, projectID contracts.ProjectID, workerID string) (ports.WorkerRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.byKey[workerKey(projectID, workerID)]
	if !ok {
		return ports.WorkerRecord{}, contracts.ErrNotFound
	}
	return w, nil
}

func (r *WorkerRepo) List(_ context.Context, projectID contracts.ProjectID) ([]ports.WorkerRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ports.WorkerRecord
	prefix := string(projectID) + "|"
	for k, w := range r.byKey {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, w)
		}
	}
	return out, nil
}

func (r *WorkerRepo) UpdateStatus(_ context.Context, projectID contracts.ProjectID, workerID, status string, version int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := workerKey(projectID, workerID)
	w, ok := r.byKey[k]
	if !ok {
		return contracts.ErrNotFound
	}
	if w.Version != version {
		return contracts.ErrConflict
	}
	w.Status = status
	w.Version++
	w.UpdatedAt = time.Now().UTC()
	r.byKey[k] = w
	return nil
}

func (r *WorkerRepo) TouchHeartbeat(_ context.Context, projectID contracts.ProjectID, workerID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := workerKey(projectID, workerID)
	w, ok := r.byKey[k]
	if !ok {
		return contracts.ErrNotFound
	}
	w.LastSeenAt = at
	w.UpdatedAt = at
	r.byKey[k] = w
	return nil
}

func (r *WorkerRepo) Revoke(ctx context.Context, projectID contracts.ProjectID, workerID string) error {
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return err
	}
	return r.UpdateStatus(ctx, projectID, workerID, "REVOKED", w.Version)
}

var _ ports.WorkerRepository = (*WorkerRepo)(nil)
