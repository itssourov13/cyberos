package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cyberos/internal/contracts"
)

// handleEventStream is a read-only Server-Sent Events endpoint.
// It never authorizes work or triggers execution.
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.hub == nil {
		writeErr(w, http.StatusServiceUnavailable, "event hub not configured")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	// Subscribe before committing headers so an event published during the
	// handshake cannot fall into the gap between response flush and subscribe.
	ch := s.hub.Subscribe(32)
	defer s.hub.Unsubscribe(ch)

	// The server-wide WriteTimeout must not terminate a long-lived stream.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Initial comment so clients know the stream is alive
	_, _ = fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case env, ok := <-ch:
			if !ok {
				return
			}
			if !s.eventVisible(r, env) {
				continue
			}
			writeSSE(w, flusher, env)
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, env contracts.EventEnvelope) {
	b, err := json.Marshal(env)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", env.Type, b)
	flusher.Flush()
}

// eventVisible limits the stream for tenant-scoped callers to events of
// projects their tenant owns. Events without a project are system-wide and
// visible only to operator (tenant-less) identities.
func (s *Server) eventVisible(r *http.Request, env contracts.EventEnvelope) bool {
	t := callerTenant(r)
	if t == "" {
		return true
	}
	if env.ProjectID == "" || s.SDK == nil || s.SDK.Catalog == nil {
		return false
	}
	owner, known := s.SDK.Catalog.ProjectTenant(env.ProjectID)
	return known && owner == t
}
