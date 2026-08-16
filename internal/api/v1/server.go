// Package v1 implements the versioned HTTP API for CYBEROS.
// Handlers are thin: they decode input, call SDK/services, and encode output.
// No business or security logic lives in handlers.
package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/sdk"
	"cyberos/pkg/version"
)

// Server is the API v1 HTTP handler root.
type Server struct {
	SDK *sdk.Client
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
	return mux
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
