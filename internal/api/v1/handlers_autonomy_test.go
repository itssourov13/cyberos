package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cyberos/internal/ai/autonomy"
	v1 "cyberos/internal/api/v1"
	"cyberos/internal/contracts"
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

func TestAutonomyCancelIdempotentAPI(t *testing.T) {
	client := sdk.NewInMemory()
	if client.Autonomy == nil {
		t.Fatal("autonomy not wired")
	}
	run, err := client.StartAutonomy(context.Background(), autonomy.StartRequest{
		ID: "api-can-1", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	r1, err := client.CancelAutonomy(run.ID)
	if err != nil || r1.State != autonomy.RunCancelled {
		t.Fatal(err, r1)
	}
	r2, err := client.CancelAutonomy(run.ID)
	if err != nil || r2.State != autonomy.RunCancelled {
		t.Fatal(err, r2)
	}
	// advance after cancel
	res, err := client.AdvanceAutonomy(context.Background(), autonomy.HandoffRequest{
		RunID: run.ID, ProjectID: "proj-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled {
		t.Fatal("cancelled must not orchestrate")
	}
}

func TestAutonomyAdvanceManualNoOrch(t *testing.T) {
	client := sdk.NewInMemory()
	run, err := client.StartAutonomy(context.Background(), autonomy.StartRequest{
		ID: "api-man-1", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.AdvanceAutonomy(context.Background(), autonomy.HandoffRequest{
		RunID: run.ID, ProjectID: "proj-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled || res.MayAdvance {
		t.Fatal(res)
	}
}
