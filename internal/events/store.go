// Package events provides an in-process, append-only event store.
// Commands and Events remain strictly separate: this package only records Events.
// No Kafka or distributed bus in Phase 1I.
package events

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// Store is an append-only, in-memory event log suitable for forensic reconstruction
// within a process. Persistence can be added later without changing the envelope.
type Store struct {
	mu     sync.RWMutex
	events []contracts.EventEnvelope
}

// NewStore constructs an empty event store.
func NewStore() *Store {
	return &Store{events: make([]contracts.EventEnvelope, 0, 64)}
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
	// Timestamp overrides clock (tests). Zero → now.
	Timestamp time.Time
}

// Append validates and appends an immutable event envelope.
// Once appended, events are never updated or deleted by this store.
func (s *Store) Append(req AppendRequest) (contracts.EventEnvelope, error) {
	if s == nil {
		return contracts.EventEnvelope{}, fmt.Errorf("%w: event store not configured", contracts.ErrInvalidArgument)
	}
	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	env := contracts.EventEnvelope{
		ID:            req.ID,
		Type:          req.Type,
		SchemaVersion: contracts.EventSchemaV1,
		Timestamp:     ts,
		Principal:     req.Principal,
		ProjectID:     req.ProjectID,
		ScopeID:       req.ScopeID,
		TaskID:        req.TaskID,
		CorrelationID: req.CorrelationID,
		CausationID:   req.CausationID,
		Payload:       req.Payload,
	}
	if err := env.Validate(); err != nil {
		return contracts.EventEnvelope{}, err
	}
	// Correlation recommended for security-critical events; enforce non-empty for audit chain.
	if env.CorrelationID == "" {
		return contracts.EventEnvelope{}, fmt.Errorf("%w: correlation_id is required", contracts.ErrInvalidArgument)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Append-only: copy payload map to avoid external mutation.
	if env.Payload != nil {
		cp := make(map[string]any, len(env.Payload))
		for k, v := range env.Payload {
			cp[k] = v
		}
		env.Payload = cp
	}
	s.events = append(s.events, env)
	return env, nil
}

// All returns a snapshot of all events in append order.
func (s *Store) All() []contracts.EventEnvelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contracts.EventEnvelope, len(s.events))
	copy(out, s.events)
	return out
}

// ByCorrelation returns events sharing a correlation ID (forensic chain).
func (s *Store) ByCorrelation(id contracts.CorrelationID) []contracts.EventEnvelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []contracts.EventEnvelope
	for _, e := range s.events {
		if e.CorrelationID == id {
			out = append(out, e)
		}
	}
	return out
}

// ByType returns events of a given type.
func (s *Store) ByType(t contracts.EventType) []contracts.EventEnvelope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []contracts.EventEnvelope
	for _, e := range s.events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// Len returns the number of stored events.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}
