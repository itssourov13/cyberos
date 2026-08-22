package v1

import (
	"net/http"

	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// GetWorkflowRun returns a single run. Read-only — no grants, no execution.
func (s *Server) GetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Workflow == nil {
		writeErr(w, http.StatusNotImplemented, "workflow not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "id required")
		return
	}
	run, err := s.SDK.Workflow.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceProject(w, r, run.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// ListWorkflowRuns lists runs for a project. Read-only.
func (s *Server) ListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Workflow == nil {
		writeErr(w, http.StatusNotImplemented, "workflow not configured")
		return
	}
	projectID := contracts.ProjectID(r.URL.Query().Get("project_id"))
	if projectID == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return
	}
	if !enforceProject(w, r, projectID) {
		return
	}
	runs, err := s.SDK.Workflow.ListByProject(projectID, 100, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runs == nil {
		runs = []*workflow.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}
