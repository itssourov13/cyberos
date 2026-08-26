// Package v1 implements the versioned HTTP API for CYBEROS.
// Handlers are thin: they decode input, call SDK/services, and encode output.
// No business or security logic lives in handlers.
package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/events"
	"cyberos/internal/sdk"
	"cyberos/pkg/version"
)

// Server is the API v1 HTTP handler root.
type Server struct {
	SDK     *sdk.Client
	hub     *events.Hub
	Auth    *auth.Middleware // optional; when set, protects API routes
	Limiter *SafetyLimiter   // Phase 16D process-local AI/autonomy rate limits
}

// NewServer constructs an API server.
func NewServer(client *sdk.Client) *Server {
	return &Server{SDK: client}
}

// Handler returns the root http.Handler for /api/v1/.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/version", s.handleVersion)
	mux.HandleFunc("/api/v1/capabilities", s.handleCapabilities)
	mux.HandleFunc("/api/v1/evaluate", s.handleEvaluate)
	mux.HandleFunc("/api/v1/grants", s.handleGrants)
	mux.HandleFunc("/api/v1/execute", s.handleExecute)
	mux.HandleFunc("/api/v1/projects", s.handleProjects)
	mux.HandleFunc("/api/v1/scopes", s.handleScopes)
	mux.HandleFunc("/api/v1/tasks", s.handleTasks)
	mux.HandleFunc("/api/v1/tasks/{id}", s.handleTaskByID)
	mux.HandleFunc("/api/v1/dashboard", s.handleDashboard)
	mux.HandleFunc("/api/v1/assessments", s.handleAssessments)
	mux.HandleFunc("/api/v1/assessments/{id}", s.handleAssessmentByID)
	mux.HandleFunc("/api/v1/assessments/{id}/plan", s.handleAssessmentPlan)
	mux.HandleFunc("/api/v1/assessments/{id}/run", s.handleAssessmentRun)
	mux.HandleFunc("/api/v1/findings", s.handleFindings)
	mux.HandleFunc("/api/v1/assets", s.handleAssets)
	mux.HandleFunc("/api/v1/risks", s.handleRisks)
	mux.HandleFunc("/api/v1/cases", s.handleCases)
	mux.HandleFunc("/api/v1/cases/{id}", s.handleCaseByID)
	mux.HandleFunc("/api/v1/cases/{id}/status", s.handleCaseStatus)
	mux.HandleFunc("/api/v1/reports", s.handleReports)
	mux.HandleFunc("/api/v1/reports/{id}", s.handleReportByID)
	mux.HandleFunc("/api/v1/reports/{id}/export", s.handleReportExport)
	mux.HandleFunc("/api/v1/autonomy/runs", s.handleAutonomyRuns)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}", s.handleAutonomyRunByID)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}/cancel", s.handleAutonomyCancel)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}/resume", s.handleAutonomyResume)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}/advance", s.handleAutonomyAdvance)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}/approve", s.handleAutonomyApprove)
	mux.HandleFunc("/api/v1/autonomy/runs/{id}/reject", s.handleAutonomyReject)
	mux.HandleFunc("/api/v1/ai/analyze", s.handleAIAnalyze)
	mux.HandleFunc("/api/v1/ai/plan", s.handleAIPlan)
	mux.HandleFunc("/api/v1/ai/apply", s.handleAIApply)
	mux.HandleFunc("/api/v1/risks/{id}", s.handleRiskByID)
	mux.HandleFunc("/api/v1/findings/{id}/risk", s.handleFindingRisk)
	mux.HandleFunc("/api/v1/assets/{id}/risks", s.handleAssetRisks)
	mux.HandleFunc("/api/v1/assessments/{id}/risk-summary", s.handleAssessmentRiskSummary)
	mux.HandleFunc("/api/v1/assets/{id}", s.handleAssetByID)
	mux.HandleFunc("/api/v1/findings/{id}", s.handleFindingByID)
	mux.HandleFunc("/api/v1/assessments/{id}/findings", s.handleAssessmentFindings)
	mux.HandleFunc("/api/v1/events/stream", s.handleEventStream)
	mux.HandleFunc("/api/v1/workflow/runs", s.ListWorkflowRuns)
	mux.HandleFunc("/api/v1/workflow/runs/{id}", s.GetWorkflowRun)
	mux.HandleFunc("POST /api/v1/workflow/runs", s.StartWorkflowRun)
	mux.HandleFunc("POST /api/v1/workflow/runs/{id}/advance", s.AdvanceWorkflowRun)
	mux.HandleFunc("POST /api/v1/workflow/runs/{id}/cancel", s.CancelWorkflowRun)
	// Web UI foundation (static)
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiFileServer()))
	mux.HandleFunc("/", s.handleRoot)
	var h http.Handler = mux
	if s.Auth != nil {
		h = s.Auth.Wrap(mux)
	}
	return h
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, version.Info())
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.SDK.ListCapabilities(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"capabilities": list})
	case http.MethodPost:
		var body struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if err := s.SDK.RegisterCapability(r.Context(), body.ID, body.Name, contracts.CapabilityType(body.Type)); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": body.ID})
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		PrincipalID     string `json:"principal_id"`
		ProjectID       string `json:"project_id"`
		ScopeID         string `json:"scope_id"`
		TaskID          string `json:"task_id"`
		CapabilityID    string `json:"capability_id"`
		AssetID         string `json:"asset_id"`
		CorrelationID   string `json:"correlation_id"`
		RequireApproval bool   `json:"require_approval"`
		RiskLevel       string `json:"risk_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.CorrelationID == "" {
		body.CorrelationID = "api-" + time.Now().UTC().Format("20060102T150405")
	}
	d, err := s.SDK.Evaluate(r.Context(), controlplane.EvaluateRequest{
		PrincipalID:     contracts.PrincipalID(body.PrincipalID),
		ProjectID:       contracts.ProjectID(body.ProjectID),
		ScopeID:         contracts.ScopeID(body.ScopeID),
		TaskID:          contracts.TaskID(body.TaskID),
		CapabilityID:    contracts.CapabilityID(body.CapabilityID),
		AssetID:         contracts.AssetID(body.AssetID),
		CorrelationID:   contracts.CorrelationID(body.CorrelationID),
		RequireApproval: body.RequireApproval,
		RiskLevelHint:   contracts.RiskLevel(body.RiskLevel),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleGrants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		PrincipalID   string `json:"principal_id"`
		ProjectID     string `json:"project_id"`
		ScopeID       string `json:"scope_id"`
		TaskID        string `json:"task_id"`
		CapabilityID  string `json:"capability_id"`
		CorrelationID string `json:"correlation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.CorrelationID == "" {
		body.CorrelationID = "api-grant-" + time.Now().UTC().Format("20060102T150405")
	}
	g, d, err := s.SDK.IssueGrant(r.Context(), controlplane.EvaluateRequest{
		PrincipalID:   contracts.PrincipalID(body.PrincipalID),
		ProjectID:     contracts.ProjectID(body.ProjectID),
		ScopeID:       contracts.ScopeID(body.ScopeID),
		TaskID:        contracts.TaskID(body.TaskID),
		CapabilityID:  contracts.CapabilityID(body.CapabilityID),
		CorrelationID: contracts.CorrelationID(body.CorrelationID),
	})
	if err != nil {
		// Denied authorization is not a 500
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":    err.Error(),
			"decision": d,
		})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"grant": g, "decision": d})
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Grant      contracts.ExecutionGrant `json:"grant"`
		Parameters map[string]any           `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	res, err := s.SDK.Execute(body.Grant, body.Parameters)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":  err.Error(),
			"result": res,
		})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui/", http.StatusFound)
}
