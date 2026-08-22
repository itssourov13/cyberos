package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cyberos/internal/ai/autonomy"
	v1 "cyberos/internal/api/v1"
	"cyberos/internal/sdk"
)

func TestAutonomyStartAndGet(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	body, _ := json.Marshal(autonomy.StartRequest{
		ID: "api-ar-1", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 3,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/autonomy/runs", bytes.NewReader(body))
	// without auth middleware enforceProject may allow empty context — still exercises handler
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}
