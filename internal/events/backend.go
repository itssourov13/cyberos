package events

import "cyberos/internal/contracts"

// Backend is the persistence surface for EventEnvelope records.
// Append-only: implementations must not update or delete existing events.
type Backend interface {
	Append(env contracts.EventEnvelope) error
	Get(id contracts.EventID) (contracts.EventEnvelope, error)
	All() ([]contracts.EventEnvelope, error)
	ByCorrelation(id contracts.CorrelationID) ([]contracts.EventEnvelope, error)
	ByType(t contracts.EventType) ([]contracts.EventEnvelope, error)
	Len() (int, error)
}

// MemoryBackend is the default in-process backend.
type MemoryBackend struct {
	events []contracts.EventEnvelope
}

// NewMemoryBackend constructs an empty memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{events: make([]contracts.EventEnvelope, 0, 64)}
}

func (m *MemoryBackend) Append(env contracts.EventEnvelope) error {
	m.events = append(m.events, env)
	return nil
}

func (m *MemoryBackend) Get(id contracts.EventID) (contracts.EventEnvelope, error) {
	for _, e := range m.events {
		if e.ID == id {
			return e, nil
		}
	}
	return contracts.EventEnvelope{}, contracts.ErrNotFound
}

func (m *MemoryBackend) All() ([]contracts.EventEnvelope, error) {
	out := make([]contracts.EventEnvelope, len(m.events))
	copy(out, m.events)
	return out, nil
}

func (m *MemoryBackend) ByCorrelation(id contracts.CorrelationID) ([]contracts.EventEnvelope, error) {
	var out []contracts.EventEnvelope
	for _, e := range m.events {
		if e.CorrelationID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *MemoryBackend) ByType(t contracts.EventType) ([]contracts.EventEnvelope, error) {
	var out []contracts.EventEnvelope
	for _, e := range m.events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *MemoryBackend) Len() (int, error) { return len(m.events), nil }
