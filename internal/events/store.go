// Package events provides an append-only event store.
// Commands and Events remain strictly separate: this package only records Events.
package events

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// Store is an append-only event log suitable for forensic reconstruction.
// Default backend is in-memory; use WithBackend for PostgreSQL durability.
type Store struct {
	mu      sync.Mutex
	backend Backend
	hub     *Hub
}

// NewStore constructs a store with an in-memory backend.
func NewStore() *Store {
	return &Store{backend: NewMemoryBackend()}
}

// WithBackend replaces the persistence backend (e.g. postgres).
func (s *Store) WithBackend(b Backend) *Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if b != nil {
		s.backend = b
	}
	return s
}

// WithHub publishes committed events to live SSE subscribers. Persistence always
// happens before publication; a failed append is never broadcast as committed.
func (s *Store) WithHub(h *Hub) *Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.hub = h
	s.mu.Unlock()
	return s
}

// AppendRequest carries fields for a new event.
type AppendRequest struct {
	ID            contracts.EventID
	Type          contracts.EventType
	Principal     *contracts.PrincipalRef
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	CorrelationID contracts.CorrelationID
	CausationID   contracts.CausationID
	Payload       map[string]any
	Timestamp     time.Time
}

// Append validates and appends an immutable event envelope.
func (s *Store) Append(req AppendRequest) (contracts.EventEnvelope, error) {
	if s == nil || s.backend == nil {
		return contracts.EventEnvelope{}, fmt.Errorf("%w: event store not configured", contracts.ErrInvalidArgument)
	}
	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	env := contracts.EventEnvelope{
		ID: req.ID, Type: req.Type, SchemaVersion: contracts.EventSchemaV1, Timestamp: ts,
		Principal: req.Principal, ProjectID: req.ProjectID, ScopeID: req.ScopeID, TaskID: req.TaskID,
		CorrelationID: req.CorrelationID, CausationID: req.CausationID, Payload: req.Payload,
	}
	if err := env.Validate(); err != nil {
		return contracts.EventEnvelope{}, err
	}
	if env.CorrelationID == "" {
		return contracts.EventEnvelope{}, fmt.Errorf("%w: correlation_id is required", contracts.ErrInvalidArgument)
	}
	if env.Payload != nil {
		cp := make(map[string]any, len(env.Payload))
		for k, v := range env.Payload {
			cp[k] = v
		}
		env.Payload = cp
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.backend.Append(env); err != nil {
		return contracts.EventEnvelope{}, err
	}
	if s.hub != nil {
		s.hub.Publish(env)
	}
	return env, nil
}

// Get returns an event by ID.
func (s *Store) Get(id contracts.EventID) (contracts.EventEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Get(id)
}

// All returns events in append order.
func (s *Store) All() []contracts.EventEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, err := s.backend.All()
	if err != nil {
		return nil
	}
	return out
}

// ByCorrelation returns events sharing a correlation ID.
func (s *Store) ByCorrelation(id contracts.CorrelationID) []contracts.EventEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, err := s.backend.ByCorrelation(id)
	if err != nil {
		return nil
	}
	return out
}

// ByType returns events of a given type.
func (s *Store) ByType(t contracts.EventType) []contracts.EventEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, err := s.backend.ByType(t)
	if err != nil {
		return nil
	}
	return out
}

// Len returns the number of stored events.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.backend.Len()
	if err != nil {
		return 0
	}
	return n
}
