package memory

import (
	"context"
	"sync"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type AssignmentRepo struct {
	mu   sync.RWMutex
	byID map[string]ports.AssignmentRecord // project|id
	byEx map[string]string                 // project|execution → id
}

func NewAssignmentRepo() *AssignmentRepo {
	return &AssignmentRepo{byID: make(map[string]ports.AssignmentRecord), byEx: make(map[string]string)}
}

func assignKey(p contracts.ProjectID, id string) string { return string(p) + "|" + id }

func (r *AssignmentRepo) Create(_ context.Context, a ports.AssignmentRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := assignKey(a.ProjectID, a.ID)
	if _, ok := r.byID[k]; ok {
		return contracts.ErrConflict
	}
	ek := assignKey(a.ProjectID, a.ExecutionID)
	if existing, ok := r.byEx[ek]; ok {
		// idempotent create for same execution
		if existing == a.ID {
			return nil
		}
		return contracts.ErrConflict
	}
	r.byID[k] = a
	r.byEx[ek] = a.ID
	return nil
}

func (r *AssignmentRepo) Get(_ context.Context, projectID contracts.ProjectID, id string) (ports.AssignmentRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byID[assignKey(projectID, id)]
	if !ok {
		return ports.AssignmentRecord{}, contracts.ErrNotFound
	}
	return a, nil
}

func (r *AssignmentRepo) GetByExecutionID(_ context.Context, projectID contracts.ProjectID, executionID string) (ports.AssignmentRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEx[assignKey(projectID, executionID)]
	if !ok {
		return ports.AssignmentRecord{}, contracts.ErrNotFound
	}
	return r.byID[assignKey(projectID, id)], nil
}

func (r *AssignmentRepo) Update(_ context.Context, a ports.AssignmentRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := assignKey(a.ProjectID, a.ID)
	cur, ok := r.byID[k]
	if !ok {
		return contracts.ErrNotFound
	}
	if cur.Version != a.Version {
		return contracts.ErrConflict
	}
	a.Version = cur.Version + 1
	r.byID[k] = a
	return nil
}

var _ ports.AssignmentRepository = (*AssignmentRepo)(nil)
