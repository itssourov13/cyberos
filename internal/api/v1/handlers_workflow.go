package v1

import (
	"encoding/json"
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

// StartWorkflowRun starts a workflow run (orchestration). Body is workflow.Definition + principal_id.
func (s *Server) StartWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Workflow == nil {
		writeErr(w, http.StatusNotImplemented, "workflow not configured")
		return
	}
	var body struct {
		Definition  workflow.Definition   `json:"definition"`
		PrincipalID contracts.PrincipalID `json:"principal_id"`
		RunID       string                `json:"run_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !enforceProject(w, r, body.Definition.ProjectID) {
		return
	}
	run, err := s.SDK.StartWorkflow(r.Context(), body.Definition, body.PrincipalID, body.RunID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// AdvanceWorkflowRun advances ready steps (each step: CP → Grant → Lifecycle).
func (s *Server) AdvanceWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Workflow == nil {
		writeErr(w, http.StatusNotImplemented, "workflow not configured")
		return
	}
	id := r.PathValue("id")
	var body struct {
		Definition workflow.Definition `json:"definition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	run, err := s.SDK.GetWorkflowRun(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceProject(w, r, run.ProjectID) {
		return
	}
	run, err = s.SDK.AdvanceWorkflow(r.Context(), id, body.Definition)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// CancelWorkflowRun cancels a run (orchestration only).
func (s *Server) CancelWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Workflow == nil {
		writeErr(w, http.StatusNotImplemented, "workflow not configured")
		return
	}
	id := r.PathValue("id")
	run, err := s.SDK.GetWorkflowRun(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceProject(w, r, run.ProjectID) {
		return
	}
	run, err = s.SDK.CancelWorkflow(id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}
