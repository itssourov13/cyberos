package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/contracts"
)

type createAssessmentBody struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ScopeID     string `json:"scope_id"`
	PrincipalID string `json:"principal_id"`
	Title       string `json:"title"`
	Target      string `json:"target"`
}

type planAssessmentBody struct {
	Steps []contracts.AssessmentStep `json:"steps"`
}

func (s *Server) handleAssessments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.SDK == nil || s.SDK.Catalog == nil {
			writeJSON(w, http.StatusOK, map[string]any{"assessments": []any{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"assessments": s.SDK.Catalog.ListAssessments()})
	case http.MethodPost:
		var body createAssessmentBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		a, err := s.SDK.CreateAssessment(body.ID, body.ProjectID, body.ScopeID, body.PrincipalID, body.Title, body.Target)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, a)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAssessmentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		a, err := s.SDK.GetAssessment(id)
		if err != nil {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, a)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAssessmentPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	var body planAssessmentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.SDK.PlanAssessment(id, body.Steps)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleAssessmentRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	a, err := s.SDK.RunAssessment(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}
