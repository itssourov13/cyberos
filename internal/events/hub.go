package events

import (
	"sync"

	"cyberos/internal/contracts"
)

// Hub fans out event envelopes to subscribers (SSE foundation).
// It is read-only observation — never authorizes or executes.
type Hub struct {
	mu   sync.RWMutex
	subs map[chan contracts.EventEnvelope]struct{}
}

// NewHub constructs an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[chan contracts.EventEnvelope]struct{})}
}

// Subscribe registers a buffered subscriber channel.
func (h *Hub) Subscribe(buffer int) chan contracts.EventEnvelope {
	if buffer < 1 {
		buffer = 16
	}
	ch := make(chan contracts.EventEnvelope, buffer)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes the channel.
func (h *Hub) Unsubscribe(ch chan contracts.EventEnvelope) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Publish delivers an event to all subscribers (non-blocking per sub).
func (h *Hub) Publish(env contracts.EventEnvelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- env:
		default:
			// drop if slow consumer
		}
	}
}

// Len returns subscriber count.
func (h *Hub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
