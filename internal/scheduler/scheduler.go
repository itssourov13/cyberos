// Package scheduler provides safe task scheduling primitives.
//
// Scheduling ≠ authorization.
// Promoting a task to PENDING does not grant execution rights.
// Execution still requires Control Plane → Grant → Tool Broker.
package scheduler

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

// Entry is a scheduled task reference held by the scheduler.
type Entry struct {
	TaskID    contracts.TaskID
	NotBefore time.Time
	Priority  int
	Enqueued  time.Time
}

// TaskStore is the minimal persistence surface the scheduler needs.
type TaskStore interface {
	GetByID(id contracts.TaskID) (*task.Task, error)
	Update(t *task.Task) error
}

// MemoryStore is an in-process TaskStore for tests and local use.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[contracts.TaskID]*task.Task
}

// NewMemoryStore constructs an empty memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[contracts.TaskID]*task.Task)}
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
	m.items[t.ID] = &cp
}

func (m *MemoryStore) GetByID(id contracts.TaskID) (*task.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	if t.Schedule != nil {
		s := *t.Schedule
		cp.Schedule = &s
	}
	return &cp, nil
}

func (m *MemoryStore) Update(t *task.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[t.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *t
	if t.Schedule != nil {
		s := *t.Schedule
		cp.Schedule = &s
	}
	m.items[t.ID] = &cp
	return nil
}

// Scheduler tracks scheduled tasks and promotes due ones to PENDING.
// It never calls the Control Plane, Issuer, or Tool Broker.
type Scheduler struct {
	mu    sync.Mutex
	store TaskStore
	index map[contracts.TaskID]Entry
	// Now overrides the clock (tests).
	Now func() time.Time
}

// New constructs a scheduler.
func New(store TaskStore) *Scheduler {
	return &Scheduler{
		store: store,
		index: make(map[contracts.TaskID]Entry),
		Now:   func() time.Time { return time.Now().UTC() },
	}
}

// ScheduleRequest carries scheduling parameters.
type ScheduleRequest struct {
	TaskID    contracts.TaskID
	NotBefore time.Time
	Priority  int
}

// Schedule attaches schedule metadata and transitions CREATED → SCHEDULED.
// It does not authorize the task.
func (s *Scheduler) Schedule(req ScheduleRequest) (*task.Task, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("%w: scheduler not configured", contracts.ErrInvalidArgument)
	}
	if req.TaskID == "" {
		return nil, fmt.Errorf("%w: task id required", contracts.ErrInvalidArgument)
	}
	if req.NotBefore.IsZero() {
		return nil, fmt.Errorf("%w: not_before required", contracts.ErrInvalidArgument)
	}

	t, err := s.store.GetByID(req.TaskID)
	if err != nil {
		return nil, err
	}
	if err := t.EnsureBindings(); err != nil {
		return nil, err
	}
	if err := t.ApplySchedule(task.ScheduleSpec{
		NotBefore: req.NotBefore.UTC(),
		Priority:  req.Priority,
	}); err != nil {
		return nil, err
	}

	// Transition into SCHEDULED when currently CREATED; if already SCHEDULED, update meta only.
	switch t.State {
	case contracts.TaskStateCreated:
		if err := t.TransitionTo(contracts.TaskStateScheduled); err != nil {
			return nil, err
		}
	case contracts.TaskStateScheduled:
		// reschedule allowed
	default:
		return nil, fmt.Errorf("%w: cannot schedule task in state %s",
			contracts.ErrInvalidStateTransition, t.State)
	}

	if err := s.store.Update(t); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.index[t.ID] = Entry{
		TaskID:    t.ID,
		NotBefore: req.NotBefore.UTC(),
		Priority:  req.Priority,
		Enqueued:  s.Now(),
	}
	s.mu.Unlock()
	return t, nil
}

// Tick promotes all due SCHEDULED tasks to PENDING.
// Returns the tasks that were promoted. Does not authorize or execute.
func (s *Scheduler) Tick() ([]*task.Task, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("%w: scheduler not configured", contracts.ErrInvalidArgument)
	}
	now := s.Now()

	s.mu.Lock()
	entries := make([]Entry, 0, len(s.index))
	for _, e := range s.index {
		entries = append(entries, e)
	}
	s.mu.Unlock()

	// Higher priority first, then earlier NotBefore.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Priority != entries[j].Priority {
			return entries[i].Priority > entries[j].Priority
		}
		return entries[i].NotBefore.Before(entries[j].NotBefore)
	})

	var promoted []*task.Task
	for _, e := range entries {
		if now.Before(e.NotBefore) {
			continue
		}
		t, err := s.store.GetByID(e.TaskID)
		if err != nil {
			continue
		}
		if t.State != contracts.TaskStateScheduled {
			s.mu.Lock()
			delete(s.index, e.TaskID)
			s.mu.Unlock()
			continue
		}
		if !t.IsDue(now) {
			continue
		}
		if err := t.TransitionTo(contracts.TaskStatePending); err != nil {
			return promoted, err
		}
		q := now
		if t.Schedule != nil {
			t.Schedule.QueuedAt = &q
		}
		if err := s.store.Update(t); err != nil {
			return promoted, err
		}
		s.mu.Lock()
		delete(s.index, e.TaskID)
		s.mu.Unlock()
		promoted = append(promoted, t)
	}
	return promoted, nil
}

// PendingScheduled returns currently indexed scheduled entries (not yet promoted).
func (s *Scheduler) PendingScheduled() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, 0, len(s.index))
	for _, e := range s.index {
		out = append(out, e)
	}
	return out
}

// Cancel removes a task from the schedule index. State transition is the caller's duty.
func (s *Scheduler) Cancel(id contracts.TaskID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.index, id)
}
