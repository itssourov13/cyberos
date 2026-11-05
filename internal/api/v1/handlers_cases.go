package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/auth"
	"cyberos/internal/cases"
	"cyberos/internal/contracts"
)

const maxCaseLinks = 500

// requirePermission enforces a role permission for the authenticated caller.
// In explicit development mode (no auth middleware) it is a no-op.
func (s *Server) requirePermission(w http.ResponseWriter, r *http.Request, perm auth.Permission) bool {
	if s.Auth == nil {
		return true
	}
	if _, ok := auth.PrincipalFromContext(r.Context()); !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if !auth.RoleFromContext(r.Context()).Allows(perm) {
		writeErr(w, http.StatusForbidden, "insufficient role for this operation")
		return false
	}
	return true
}

// actorID returns the authenticated principal id used for attribution.
func (s *Server) actorID(r *http.Request) contracts.PrincipalID {
	if p, ok := auth.PrincipalFromContext(r.Context()); ok {
		return p.ID
	}
	return "dev"
}

type createCaseBody struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	AssessmentID string                 `json:"assessment_id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	FindingIDs   []contracts.FindingID  `json:"finding_ids"`
	AssetIDs     []contracts.AssetID    `json:"asset_ids"`
	Priority     contracts.RiskPriority `json:"priority"`
	// tenant_id / created_by in a payload are deliberately NOT modeled: ownership
	// and attribution come only from server-side identity and the project record.
}

// loadCase loads a case and authorizes the caller against its stored ownership.
func (s *Server) loadCase(w http.ResponseWriter, r *http.Request, id string) (*cases.Case, bool) {
	if s.SDK == nil || s.SDK.Cases == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return nil, false
	}
	c, err := s.SDK.Cases.Get(id)
	if err != nil || c == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return nil, false
	}
	if !enforceObject(w, r, c.TenantID, c.ProjectID) {
		return nil, false
	}
	return c, true
}

func (s *Server) handleCases(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Cases == nil {
		writeJSON(w, http.StatusOK, map[string]any{"cases": []any{}})
		return
	}
	switch r.Method {
	case http.MethodGet:
		project, ok := requireProjectScope(w, r)
		if !ok {
			return
		}
		list := []*cases.Case{}
		if project != "" {
			list = s.SDK.Cases.ListByProject(project)
		}
		ct := callerTenant(r)
		out := make([]*cases.Case, 0, len(list))
		for _, c := range list {
			if ct != "" && c.TenantID != ct {
				continue
			}
			out = append(out, c)
		}
		writeJSON(w, http.StatusOK, map[string]any{"cases": out})
	case http.MethodPost:
		if !s.requirePermission(w, r, auth.PermWrite) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var body createCaseBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		project := contracts.ProjectID(body.ProjectID)
		if project == "" {
			writeErr(w, http.StatusBadRequest, "project_id required")
			return
		}
		if !enforceProject(w, r, project) {
			return
		}
		actor, ok := s.bindPrincipal(w, r, "")
		if !ok {
			return
		}
		if actor == "" {
			actor = s.actorID(r)
		}
		// Ownership comes from the canonical project record, never the payload.
		tenant, known := projectOwner(r, project)
		if !known {
			writeErr(w, http.StatusBadRequest, "unknown project")
			return
		}
		if msg := s.validateCaseLinks(project, body); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		c, err := s.SDK.Cases.Create(cases.CreateRequest{
			ID: body.ID, TenantID: tenant, ProjectID: project, AssessmentID: body.AssessmentID,
			Title: body.Title, Description: body.Description,
			FindingIDs: body.FindingIDs, AssetIDs: body.AssetIDs,
			CreatedBy: actor, Priority: body.Priority,
		})
		if err != nil {
			writeErr(w, statusForErr(err), publicErr(err))
			return
		}
		writeJSON(w, http.StatusCreated, c)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// validateCaseLinks checks that every linked assessment/finding/asset belongs
// to the case's project according to the stored records. It returns an error
// message or "".
func (s *Server) validateCaseLinks(project contracts.ProjectID, b createCaseBody) string {
	if len(b.FindingIDs) > maxCaseLinks || len(b.AssetIDs) > maxCaseLinks {
		return "too many linked items"
	}
	if b.AssessmentID != "" {
		a, err := s.SDK.GetAssessment(b.AssessmentID)
		if err != nil || a == nil || a.ProjectID != project {
			return "assessment does not belong to project"
		}
	}
	for _, fid := range b.FindingIDs {
		if s.SDK.Findings == nil {
			return "finding does not belong to project"
		}
		f, err := s.SDK.Findings.Get(fid)
		if err != nil || f == nil || f.ProjectID != project {
			return "finding does not belong to project"
		}
	}
	for _, aid := range b.AssetIDs {
		if s.SDK.Assets == nil {
			return "asset does not belong to project"
		}
		a, err := s.SDK.Assets.Get(aid)
		if err != nil || a == nil || a.ProjectID != project {
			return "asset does not belong to project"
		}
	}
	return ""
}

func (s *Server) handleCaseByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c, ok := s.loadCase(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleCaseHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	c, ok := s.loadCase(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	h, err := s.SDK.Cases.History(c.ID)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if h == nil {
		h = []cases.StatusChange{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"case_id": c.ID, "history": h})
}

func (s *Server) handleCaseStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Authorize against the stored case before reading or applying anything.
	c, ok := s.loadCase(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	if !s.requirePermission(w, r, auth.PermCaseTransition) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	var body struct {
		Status contracts.CaseStatus `json:"status"`
		Reason string               `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(body.Reason) > 1000 {
		writeErr(w, http.StatusBadRequest, "reason too long")
		return
	}
	// Risk acceptance is a sensitive transition and needs a stronger permission.
	if body.Status == contracts.CaseStatusAccepted && !s.requirePermission(w, r, auth.PermCaseAccept) {
		return
	}
	actor, ok := s.bindPrincipal(w, r, "")
	if !ok {
		return
	}
	if actor == "" {
		actor = s.actorID(r)
	}
	out, err := s.SDK.Cases.TransitionStatusBy(c.ID, body.Status, actor, body.Reason)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Reports == nil {
		writeJSON(w, http.StatusOK, map[string]any{"reports": []any{}})
		return
	}
	if r.Method == http.MethodGet {
		list := s.SDK.Reports.List()
		ct := callerTenant(r)
		proj := requestedProject(r)
		if ct != "" && proj == "" {
			writeErr(w, http.StatusForbidden, "project_id required for tenant-scoped list")
			return
		}
		out := list[:0]
		for _, rep := range list {
			if proj != "" && rep.ProjectID != proj {
				continue
			}
			if ct != "" && rep.TenantID != "" && rep.TenantID != ct {
				continue
			}
			out = append(out, rep)
		}
		writeJSON(w, http.StatusOK, map[string]any{"reports": out})
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
	if !enforceResource(w, r, rep.TenantID, rep.ProjectID) {
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
	rep, err := s.SDK.Reports.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceResource(w, r, rep.TenantID, rep.ProjectID) {
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
