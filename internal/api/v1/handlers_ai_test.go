package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cyberos/internal/ai"
	v1 "cyberos/internal/api/v1"
	"cyberos/internal/sdk"
)

func TestAIAnalyzeEndpoint(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	body, _ := json.Marshal(ai.BrainRequest{
		ProjectID: "proj-a", ScopeID: "s", Objective: "summarize findings",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
}

func TestAIPlanEndpoint(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	body, _ := json.Marshal(ai.BrainRequest{
		ProjectID: "proj-a", ScopeID: "s", Objective: "port scan",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/plan", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
}
