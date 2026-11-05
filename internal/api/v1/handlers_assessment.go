package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/assessment"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
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

// loadAssessment loads the stored assessment and authorizes the caller against
// its canonical project/tenant before anything is returned or mutated. Missing
// and not-owned assessments are indistinguishable (404).
func (s *Server) loadAssessment(w http.ResponseWriter, r *http.Request, id string) (*assessment.Assessment, bool) {
	if id == "" {
		writeErr(w, http.StatusBadRequest, "id required")
		return nil, false
	}
	if s.SDK == nil || s.SDK.Assessment == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return nil, false
	}
	a, err := s.SDK.GetAssessment(id)
	if err != nil || a == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return nil, false
	}
	if !enforceObject(w, r, "", a.ProjectID) {
		return nil, false
	}
	return a, true
}

func (s *Server) handleAssessments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.SDK == nil || s.SDK.Catalog == nil {
			writeJSON(w, http.StatusOK, map[string]any{"assessments": []any{}})
			return
		}
		project, ok := requireProjectScope(w, r)
		if !ok {
			return
		}
		out := make([]sdk.AssessmentView, 0)
		for _, a := range s.SDK.Catalog.ListAssessments() {
			if project != "" && contracts.ProjectID(a.ProjectID) != project {
				continue
			}
			if !resourceVisible(r, contracts.ProjectID(a.ProjectID)) {
				continue
			}
			out = append(out, a)
		}
		writeJSON(w, http.StatusOK, map[string]any{"assessments": out})
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var body createAssessmentBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		principal, ok := s.bindPrincipal(w, r, contracts.PrincipalID(body.PrincipalID))
		if !ok {
			return
		}
		if !enforceProject(w, r, contracts.ProjectID(body.ProjectID)) {
			return
		}
		if !s.scopeBelongsToProject(contracts.ProjectID(body.ProjectID), contracts.ScopeID(body.ScopeID)) {
			writeErr(w, http.StatusBadRequest, "scope does not belong to project")
			return
		}
		a, err := s.SDK.CreateAssessment(body.ID, body.ProjectID, body.ScopeID, string(principal), body.Title, body.Target)
		if err != nil {
			writeErr(w, statusForErr(err), publicErr(err))
			return
		}
		writeJSON(w, http.StatusCreated, a)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// scopeBelongsToProject validates a scope against the canonical catalog.
func (s *Server) scopeBelongsToProject(p contracts.ProjectID, sc contracts.ScopeID) bool {
	if s.SDK == nil || s.SDK.Catalog == nil {
		return false
	}
	for _, v := range s.SDK.Catalog.ListScopes(p) {
		if contracts.ScopeID(v.ID) == sc {
			return true
		}
	}
	return false
}

func (s *Server) handleAssessmentByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	a, ok := s.loadAssessment(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleAssessmentPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if _, ok := s.loadAssessment(w, r, id); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body planAssessmentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	a, err := s.SDK.PlanAssessment(id, body.Steps)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
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
	if _, ok := s.loadAssessment(w, r, id); !ok {
		return
	}
	a, err := s.SDK.RunAssessment(r.Context(), id)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, a)
}
