package v1

import (
	"net/http"
	"strconv"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/auth"
)

const (
	auditDefaultLimit = 100
	auditMaxLimit     = 500
)

// handleAuditList serves GET /api/v1/audit — a read-only, newest-first feed of
// audit entries.
//
// Authorization:
//   - requires audit.read (analyst and above);
//   - tenant-scoped callers must name a project they own (requireProjectScope)
//     and only ever see entries recorded against that project; entries with no
//     project are platform-level and are visible to the operator only;
//   - the operator may omit project_id to see everything.
//
// Metadata is deliberately not exposed: it is free-form and may carry values
// that were never meant for a browser.
func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.requirePermission(w, r, auth.PermAuditRead) {
		return
	}
	project, ok := requireProjectScope(w, r)
	if !ok {
		return
	}
	limit := auditDefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeErr(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = min(n, auditMaxLimit)
	}
	if s.SDK == nil || s.SDK.Audit == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []any{}})
		return
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeErr(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}
	if project == "" && callerTenant(r) != "" {
		writeErr(w, http.StatusForbidden, "project_id required")
		return
	}
	picked, err := s.SDK.Audit.List(project, limit, offset)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "audit store unavailable")
		return
	}
	out := make([]map[string]any, 0, len(picked))
	for _, e := range picked {
		out = append(out, auditView(e))
	}
	nextOffset := offset + len(picked)
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": out, "limit": limit, "offset": offset, "next_offset": nextOffset,
	})
}

func auditView(e audit.Entry) map[string]any {
	actor := ""
	if e.Principal != nil {
		actor = string(e.Principal.ID)
	}
	return map[string]any{
		"id":             e.ID,
		"action":         e.Action,
		"at":             e.Timestamp.UTC().Format(time.RFC3339),
		"actor":          actor,
		"project_id":     e.ProjectID,
		"scope_id":       e.ScopeID,
		"task_id":        e.TaskID,
		"decision_id":    e.DecisionID,
		"grant_id":       e.GrantID,
		"correlation_id": e.CorrelationID,
		"outcome":        e.Outcome,
		"reason":         e.Reason,
	}
}
