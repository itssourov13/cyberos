package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/ai"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// handleAIAnalyze POST /api/v1/ai/analyze — untrusted analysis only.
func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil {
		writeErr(w, http.StatusNotImplemented, "sdk not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "ai") {
		return
	}
	if !s.requirePermission(w, r, auth.PermWrite) {
		return
	}
	var req ai.BrainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !enforceProject(w, r, req.ProjectID) {
		return
	}
	resp, err := s.SDK.Analyze(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, sanitizeProviderErr(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, aiSafeResponse(resp))
}

// handleAIPlan POST /api/v1/ai/plan — untrusted plan + ValidatePlan.
func (s *Server) handleAIPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil {
		writeErr(w, http.StatusNotImplemented, "sdk not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "ai") {
		return
	}
	if !s.requirePermission(w, r, auth.PermWrite) {
		return
	}
	var req ai.BrainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !enforceProject(w, r, req.ProjectID) {
		return
	}
	resp, err := s.SDK.Plan(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, sanitizeProviderErr(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, aiSafeResponse(resp))
}

// handleAIApply POST /api/v1/ai/apply — plan → workflow definition only.
func (s *Server) handleAIApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil {
		writeErr(w, http.StatusNotImplemented, "sdk not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "ai") {
		return
	}
	if !s.requirePermission(w, r, auth.PermWrite) {
		return
	}
	var body struct {
		Plan ai.Plan `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !enforceProject(w, r, body.Plan.ProjectID) {
		return
	}
	if !s.requirePermission(w, r, auth.PermExecute) {
		return
	}
	if !s.scopeBelongsToProject(body.Plan.ProjectID, body.Plan.ScopeID) {
		writeErr(w, http.StatusBadRequest, "scope does not belong to project")
		return
	}
	def, err := s.SDK.ApplyPlan(r.Context(), body.Plan)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workflow_definition_id": def.ID,
		"project_id":             def.ProjectID,
		"steps":                  len(def.Steps),
	})
}

func aiSafeResponse(resp ai.BrainResponse) map[string]any {
	out := map[string]any{
		"status":        resp.Status,
		"provider_meta": resp.ProviderMeta,
	}
	if resp.Error != "" {
		out["error"] = sanitizeProviderErr(resp.Error)
	}
	if resp.Plan != nil {
		out["plan"] = resp.Plan
	}
	if resp.Analysis != nil {
		out["analysis"] = resp.Analysis
	}
	return out
}

func sanitizeProviderErr(s string) string {
	// Never echo credential material; strip common secret patterns.
	for _, bad := range []string{"Bearer ", "sk-", "api_key=", "Authorization:"} {
		if len(s) > 0 && containsCI(s, bad) {
			return "provider error"
		}
	}
	return s
}

func containsCI(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		(func() bool {
			ls, lsub := toLower(s), toLower(sub)
			for i := 0; i+len(lsub) <= len(ls); i++ {
				if ls[i:i+len(lsub)] == lsub {
					return true
				}
			}
			return false
		})())
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// silence unused contracts import if needed
var _ = contracts.ProjectID("")
