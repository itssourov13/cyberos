// Package runner provides durable task-runner foundation primitives.
//
// Claiming or recovering a task does NOT authorize execution.
// Workers must still pass Scope → Principal → Control Plane → Grant → Broker → Sandbox.
package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/storage/ports"
)

// WorkerID identifies a runner instance.
type WorkerID string

// Claim is a temporary exclusive hold on a task for processing.
type Claim struct {
	TaskID     contracts.TaskID
	WorkerID   WorkerID
	LeaseUntil time.Time
	Task       *task.Task
}

// Store is the persistence surface required by the runner.
type Store interface {
	GetByID(ctx context.Context, id contracts.TaskID) (*task.Task, error)
	Update(ctx context.Context, t *task.Task) error
	ListByState(ctx context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error)
	ListDueScheduled(ctx context.Context, now time.Time, limit int) ([]*task.Task, error)
	// TryClaimAtomically claims up to limit PENDING tasks for worker with exclusive lease.
	// Implementations must ensure two workers cannot claim the same task.
	TryClaimAtomically(ctx context.Context, worker WorkerID, leaseUntil time.Time, now time.Time, limit int) ([]*task.Task, error)
	// ClearExpiredLeases releases leases past now so tasks can be reclaimed.
	ClearExpiredLeases(ctx context.Context, now time.Time) (int, error)
}

// PortStore adapts ports.TaskRepository to Store.
type PortStore struct {
	Repo ports.TaskRepository
}

func (p PortStore) GetByID(ctx context.Context, id contracts.TaskID) (*task.Task, error) {
	return p.Repo.GetByID(ctx, id)
}
func (p PortStore) Update(ctx context.Context, t *task.Task) error {
	return p.Repo.Update(ctx, t)
}
func (p PortStore) ListByState(ctx context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error) {
	return p.Repo.ListByState(ctx, state, limit, offset)
}
func (p PortStore) ListDueScheduled(ctx context.Context, now time.Time, limit int) ([]*task.Task, error) {
	return p.Repo.ListDueScheduled(ctx, now, limit)
}

// MemoryStore is an in-process durable-style store for tests.
// Lease table is separate so claim exclusivity does not depend on metadata alone.
type MemoryStore struct {
	mu     sync.RWMutex
	items  map[contracts.TaskID]*task.Task
	leases map[contracts.TaskID]struct {
		Worker WorkerID
		Until  time.Time
	}
}

// NewMemoryStore constructs an empty memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[contracts.TaskID]*task.Task),
		leases: make(map[contracts.TaskID]struct {
			Worker WorkerID
			Until  time.Time
		}),
	}
}

// Put inserts or replaces a task.
func (m *MemoryStore) Put(t *task.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	if t.Schedule != nil {
		s := *t.Schedule
		cp.Schedule = &s
	}
	if t.Retry != nil {
		r := *t.Retry
		cp.Retry = &r
	}
	if t.DependsOn != nil {
		cp.DependsOn = append([]contracts.TaskID{}, t.DependsOn...)
	}
	m.items[t.ID] = &cp
}

func (m *MemoryStore) GetByID(_ context.Context, id contracts.TaskID) (*task.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return cloneTask(t), nil
}

func (m *MemoryStore) Update(_ context.Context, t *task.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[t.ID]; !ok {
		return contracts.ErrNotFound
	}
	m.items[t.ID] = cloneTask(t)
	return nil
}

func (m *MemoryStore) ListByState(_ context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*task.Task
	i := 0
	for _, t := range m.items {
		if t.State != state {
			continue
		}
		if i < offset {
			i++
			continue
		}
		out = append(out, cloneTask(t))
		if limit > 0 && len(out) >= limit {
			break
		}
		i++
	}
	return out, nil
}

func (m *MemoryStore) ListDueScheduled(_ context.Context, now time.Time, limit int) ([]*task.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*task.Task
	for _, t := range m.items {
		if t.State == contracts.TaskStateScheduled && t.IsDue(now) {
			out = append(out, cloneTask(t))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func cloneTask(t *task.Task) *task.Task {
	cp := *t
	if t.Schedule != nil {
		s := *t.Schedule
		cp.Schedule = &s
	}
	if t.Retry != nil {
		r := *t.Retry
		cp.Retry = &r
	}
	if t.DependsOn != nil {
		cp.DependsOn = append([]contracts.TaskID{}, t.DependsOn...)
	}
	return &cp
}

// Runner promotes due schedules and claims PENDING work for workers.
// It never calls Control Plane, Issuer, Broker, or Executors.
type Runner struct {
	store    Store
	leaseTTL time.Duration
	Now      func() time.Time
	mu       sync.Mutex
	leases   map[contracts.TaskID]Claim // in-process lease index (supplements store metadata)
}

// New constructs a runner.
func New(store Store, leaseTTL time.Duration) *Runner {
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	return &Runner{
		store:    store,
		leaseTTL: leaseTTL,
		Now:      func() time.Time { return time.Now().UTC() },
		leases:   make(map[contracts.TaskID]Claim),
	}
}

// PromoteDueScheduled moves due SCHEDULED tasks to PENDING.
// This is not authorization.
func (r *Runner) PromoteDueScheduled(ctx context.Context, limit int) ([]*task.Task, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("%w: runner not configured", contracts.ErrInvalidArgument)
	}
	now := r.Now()
	due, err := r.store.ListDueScheduled(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	var out []*task.Task
	for _, t := range due {
		if err := t.TransitionTo(contracts.TaskStatePending); err != nil {
			continue
		}
		if t.Schedule != nil {
			q := now
			t.Schedule.QueuedAt = &q
		}
		if err := r.store.Update(ctx, t); err != nil {
			return out, err
		}
		out = append(out, t)
	}
	return out, nil
}

// ClaimPending leases PENDING tasks for a worker using atomic store claiming.
// Claiming does not authorize or execute.
func (r *Runner) ClaimPending(ctx context.Context, worker WorkerID, limit int) ([]Claim, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("%w: runner not configured", contracts.ErrInvalidArgument)
	}
	if worker == "" {
		return nil, fmt.Errorf("%w: worker id required", contracts.ErrInvalidArgument)
	}
	if limit <= 0 {
		limit = 1
	}
	now := r.Now()
	leaseUntil := now.Add(r.leaseTTL)
	claimed, err := r.store.TryClaimAtomically(ctx, worker, leaseUntil, now, limit)
	if err != nil {
		return nil, err
	}
	var claims []Claim
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range claimed {
		claim := Claim{TaskID: t.ID, WorkerID: worker, LeaseUntil: leaseUntil, Task: t}
		r.leases[t.ID] = claim
		claims = append(claims, claim)
	}
	return claims, nil
}

// RecoverExpiredLeases clears expired leases in store and local index.
// Does not change task state to AUTHORIZED/RUNNING and does not execute.
func (r *Runner) RecoverExpiredLeases() int {
	now := r.Now()
	n := 0
	if r.store != nil {
		if cleared, err := r.store.ClearExpiredLeases(context.Background(), now); err == nil {
			n += cleared
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, c := range r.leases {
		if !c.LeaseUntil.After(now) {
			delete(r.leases, id)
			n++
		}
	}
	return n
}

// Release removes a lease after the worker finishes (success or fail path).
func (r *Runner) Release(taskID contracts.TaskID) {
	r.mu.Lock()
	delete(r.leases, taskID)
	r.mu.Unlock()
}

// ActiveLeases returns a snapshot of current leases.
func (r *Runner) ActiveLeases() []Claim {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Claim, 0, len(r.leases))
	for _, c := range r.leases {
		out = append(out, c)
	}
	return out
}

func (m *MemoryStore) TryClaimAtomically(_ context.Context, worker WorkerID, leaseUntil time.Time, now time.Time, limit int) ([]*task.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*task.Task
	for id, tsk := range m.items {
		if limit > 0 && len(out) >= limit {
			break
		}
		if tsk.State != contracts.TaskStatePending {
			continue
		}
		if lease, ok := m.leases[id]; ok && lease.Until.After(now) {
			continue
		}
		cp := *tsk
		if tsk.Schedule != nil {
			s := *tsk.Schedule
			cp.Schedule = &s
		}
		if cp.Metadata == nil {
			cp.Metadata = map[string]string{}
		} else {
			md := map[string]string{}
			for k, v := range tsk.Metadata {
				md[k] = v
			}
			cp.Metadata = md
		}
		cp.Metadata["runner_worker"] = string(worker)
		cp.Metadata["runner_lease_until"] = leaseUntil.Format(time.RFC3339)
		m.items[id] = &cp
		m.leases[id] = struct {
			Worker WorkerID
			Until  time.Time
		}{Worker: worker, Until: leaseUntil}
		out = append(out, &cp)
	}
	return out, nil
}

func (m *MemoryStore) ClearExpiredLeases(_ context.Context, now time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, lease := range m.leases {
		if lease.Until.After(now) {
			continue
		}
		delete(m.leases, id)
		if tsk, ok := m.items[id]; ok && tsk.Metadata != nil {
			cp := *tsk
			cp.Metadata = map[string]string{}
			for k, v := range tsk.Metadata {
				if k == "runner_worker" || k == "runner_lease_until" {
					continue
				}
				cp.Metadata[k] = v
			}
			m.items[id] = &cp
		}
		n++
	}
	return n, nil
}
