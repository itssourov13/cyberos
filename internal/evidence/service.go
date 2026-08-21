// Package evidence provides services for recording structured Evidence
// produced by executions under valid grants.
package evidence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
	"cyberos/internal/storage/ports"
)

// Service records and retrieves Evidence.
// Phase 1H uses an in-memory store; persistence ports can be added later.
type Service struct {
	mu    sync.RWMutex
	items map[contracts.EvidenceID]*domainevidence.Evidence
	repo  ports.EvidenceRepository // optional durable backend (Phase 8A)
}

// NewService constructs an evidence service.
func NewService() *Service {
	return &Service{items: make(map[contracts.EvidenceID]*domainevidence.Evidence)}
}

// WithRepository attaches optional durable persistence. When set, persist errors propagate.
func (s *Service) WithRepository(repo ports.EvidenceRepository) *Service {
	if s != nil {
		s.repo = repo
	}
	return s
}

func (s *Service) persist(e *domainevidence.Evidence) error {
	if s == nil || s.repo == nil || e == nil {
		return nil
	}
	return s.repo.Upsert(context.Background(), e)
}

// RecordFromExecution creates Evidence from an execution result and grant context.
// Every Evidence item must be linked to principal, scope, task, capability, and grant.
func (s *Service) RecordFromExecution(
	result contracts.ExecutionResult,
	grant contracts.ExecutionGrant,
	evidenceID contracts.EvidenceID,
) (*domainevidence.Evidence, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: evidence service not configured", contracts.ErrInvalidArgument)
	}
	if err := grant.Validate(); err != nil {
		return nil, fmt.Errorf("%w: grant: %v", contracts.ErrInvalidArgument, err)
	}
	if result.Status != contracts.ExecutionStatusSucceeded &&
		result.Status != contracts.ExecutionStatusFailed {
		// Still allow recording failed executions as evidence of the attempt.
	}

	ev, err := domainevidence.New(
		evidenceID,
		grant.TaskID,
		grant.ID,
		grant.CapabilityID,
		grant.Principal,
		grant.ProjectID,
		grant.ScopeID,
	)
	if err != nil {
		return nil, err
	}
	ev.ContentType = "application/json"
	ev.Payload = result.Output
	if ev.Payload == nil {
		ev.Payload = map[string]any{}
	}
	ev.Payload["execution_status"] = string(result.Status)
	ev.Payload["executor_id"] = result.ExecutorID
	if result.ErrorMessage != "" {
		ev.Payload["error"] = result.ErrorMessage
	}
	ev.CorrelationID = result.CorrelationID
	if ev.CorrelationID == "" {
		ev.CorrelationID = grant.CorrelationID
	}
	if !result.FinishedAt.IsZero() {
		ev.ProducedAt = result.FinishedAt.UTC()
	}

	// Simple integrity digest over key fields (not cryptographic chain yet).
	ev.Digest = fmt.Sprintf("sha256-stub:%s:%s:%s", ev.ID, ev.GrantID, ev.TaskID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[ev.ID]; exists {
		return nil, contracts.ErrConflict
	}
	cp := *ev
	s.items[ev.ID] = &cp
	if err := s.persist(&cp); err != nil {
		delete(s.items, ev.ID)
		return nil, err
	}
	return &cp, nil
}

// Get returns evidence by ID.
func (s *Service) Get(id contracts.EvidenceID) (*domainevidence.Evidence, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: evidence service not configured", contracts.ErrInvalidArgument)
	}
	s.mu.RLock()
	ev, ok := s.items[id]
	if ok {
		cp := *ev
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()
	if s.repo != nil {
		return s.repo.GetByID(context.Background(), id)
	}
	return nil, contracts.ErrNotFound
}

// ListByTask returns evidence for a task.
func (s *Service) ListByTask(taskID contracts.TaskID) []*domainevidence.Evidence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domainevidence.Evidence
	for _, ev := range s.items {
		if ev.TaskID == taskID {
			cp := *ev
			out = append(out, &cp)
		}
	}
	return out
}

// EnsureLinked verifies evidence is fully linked to execution context.
func EnsureLinked(ev *domainevidence.Evidence) error {
	if ev == nil {
		return contracts.ErrInvalidArgument
	}
	if err := ev.Validate(); err != nil {
		return err
	}
	if ev.GrantID == "" || ev.TaskID == "" || ev.ScopeID == "" ||
		ev.ProjectID == "" || ev.CapabilityID == "" || ev.Principal.ID == "" {
		return fmt.Errorf("%w: evidence missing required execution context links", contracts.ErrInvalidArgument)
	}
	return nil
}

// ProducedAt returns production time (helper for tests).
func ProducedAt(ev *domainevidence.Evidence) time.Time {
	if ev == nil {
		return time.Time{}
	}
	return ev.ProducedAt
}
