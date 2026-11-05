package v1

import (
	"net/http"

	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
)

func (s *Server) handleRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	assessment := r.URL.Query().Get("assessment_id")
	if assessment != "" {
		a, ok := s.loadAssessment(w, r, assessment)
		if !ok {
			return
		}
		list := s.SDK.RiskPrio.ListByAssessment(assessment)
		filtered := make([]*riskprio.Evaluation, 0, len(list))
		for _, ev := range list {
			if ev.ProjectID == a.ProjectID {
				filtered = append(filtered, ev)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"risks": filtered})
		return
	}
	project, ok := requireProjectScope(w, r)
	if !ok {
		return
	}
	if project == "" {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": s.SDK.RiskPrio.ListByProject(project)})
}

func (s *Server) handleRiskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ev, err := s.SDK.RiskPrio.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceObject(w, r, "", ev.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleFindingRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ev, err := s.SDK.RiskPrio.GetByFinding(contracts.FindingID(id))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceObject(w, r, "", ev.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleAssessmentRiskSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	// Authorize against the stored assessment before any risk data is read.
	if _, ok := s.loadAssessment(w, r, id); !ok {
		return
	}
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"summary": map[string]any{}})
		return
	}
	writeJSON(w, http.StatusOK, s.SDK.RiskPrio.SummarizeAssessment(id))
}

func (s *Server) handleAssetRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	// Resolve the asset first and authorize its canonical project.
	if s.SDK == nil || s.SDK.Assets == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	asset, aerr := s.SDK.Assets.Get(contracts.AssetID(id))
	if aerr != nil || asset == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, "", asset.ProjectID) {
		return
	}
	if s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	// Defense in depth: never return evaluations that belong to another project.
	risks := s.SDK.RiskPrio.ListByAsset(contracts.AssetID(id))
	out := make([]*riskprio.Evaluation, 0, len(risks))
	for _, ev := range risks {
		if ev.ProjectID == asset.ProjectID {
			out = append(out, ev)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": out})
}
