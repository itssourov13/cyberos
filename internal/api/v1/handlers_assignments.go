package v1

import (
	"net/http"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution"
)

func (s *Server) registerAssignmentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{project}/assignments", s.handleAssignmentList)
}

// handleAssignmentList serves a read-only, newest-first view of a project's
// execution assignments. It never exposes grants or secrets (the record holds
// only the grant id), and tenant callers only reach projects they own.
func (s *Server) handleAssignmentList(w http.ResponseWriter, r *http.Request) {
	if s.Assignments == nil {
		writeErr(w, http.StatusNotImplemented, "assignments not configured")
		return
	}
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	list, err := s.Assignments.List(r.Context(), project, 200)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	ct := callerTenant(r)
	views := make([]map[string]any, 0, len(list))
	for _, a := range list {
		if ct != "" && a.TenantID != ct {
			continue
		}
		views = append(views, assignmentView(a))
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "assignments": views})
}

func assignmentView(a execution.ExecutionAssignment) map[string]any {
	ts := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"assignment_id": a.ID, "execution_id": a.ExecutionID, "worker_id": a.WorkerID,
		"host_id": a.HostID, "grant_id": a.GrantID, "status": a.Status,
		"attempt": a.Attempt, "max_attempts": a.MaxAttempts,
		"lease_until": ts(a.LeaseUntil), "created_at": ts(a.CreatedAt), "completed_at": ts(a.CompletedAt),
		"result_status": a.ResultStatus, "result_hash": a.ResultHash,
	}
}
