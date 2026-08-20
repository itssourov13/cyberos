package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/cases"
	"cyberos/internal/contracts"
)

func (s *Server) handleCases(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Cases == nil {
		writeJSON(w, http.StatusOK, map[string]any{"cases": []any{}})
		return
	}
	switch r.Method {
	case http.MethodGet:
		project := r.URL.Query().Get("project_id")
		writeJSON(w, http.StatusOK, map[string]any{"cases": s.SDK.Cases.ListByProject(contracts.ProjectID(project))})
	case http.MethodPost:
		var req cases.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		c, err := s.SDK.Cases.Create(req)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, c)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCaseByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.Cases == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c, err := s.SDK.Cases.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleCaseStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	var body struct {
		Status contracts.CaseStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	c, err := s.SDK.Cases.TransitionStatus(id, body.Status)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Reports == nil {
		writeJSON(w, http.StatusOK, map[string]any{"reports": []any{}})
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"reports": s.SDK.Reports.List()})
		return
	}
	writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) handleReportByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.Reports == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rep, err := s.SDK.Reports.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleReportExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	format := contracts.ReportFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = contracts.ReportFormatJSON
	}
	if s.SDK == nil || s.SDK.Reports == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	b, err := s.SDK.Reports.Export(id, format)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if format == contracts.ReportFormatMarkdown {
		w.Header().Set("Content-Type", "text/markdown")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
