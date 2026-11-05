package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/sdk"
)

func TestExecutePersistsEvidenceAndReturnsIdempotentResult(t *testing.T) {
	client := testClient(t)
	client.Catalog.UpsertTask(sdk.TaskView{
		ID: "task-execute-1", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "cap-1", Title: "API execution test",
	})
	grant, decision, err := client.Issuer.IssueAfterEvaluate(context.Background(), client.Control, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: "task-execute-1", CapabilityID: "cap-1", CorrelationID: "execute-idempotency-test",
	})
	if err != nil || !decision.Allowed() {
		t.Fatalf("issue grant: decision=%+v err=%v", decision, err)
	}

	body, err := json.Marshal(map[string]any{
		"grant":      grant.ExecutionGrant,
		"parameters": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(client).Handler()
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/execute", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, req)
		return rr
	}

	first := request()
	if first.Code != http.StatusOK {
		t.Fatalf("first execute: %d %s", first.Code, first.Body.String())
	}
	var firstResult contracts.ExecutionResult
	if err := json.Unmarshal(first.Body.Bytes(), &firstResult); err != nil {
		t.Fatal(err)
	}
	firstEvidence, ok := firstResult.Output["evidence_id"].(string)
	if !ok || firstEvidence == "" {
		t.Fatalf("first execution missing evidence link: %+v", firstResult.Output)
	}
	firstExecution, ok := firstResult.Output["execution_id"].(string)
	if !ok || firstExecution == "" {
		t.Fatalf("first execution missing lifecycle record id: %+v", firstResult.Output)
	}
	if _, err := client.Evidence.Get(contracts.EvidenceID(firstEvidence)); err != nil {
		t.Fatalf("evidence was not persisted: %v", err)
	}
	records := client.Lifecycle.ListByTask("task-execute-1")
	if len(records) != 1 {
		t.Fatalf("expected one lifecycle execution record, got %d", len(records))
	}

	second := request()
	if second.Code != http.StatusOK {
		t.Fatalf("idempotent retry: %d %s", second.Code, second.Body.String())
	}
	var secondResult contracts.ExecutionResult
	if err := json.Unmarshal(second.Body.Bytes(), &secondResult); err != nil {
		t.Fatal(err)
	}
	if secondResult.Output["evidence_id"] != firstEvidence || secondResult.Output["execution_id"] != firstExecution {
		t.Fatalf("retry did not return the original evidence-backed result: first=%+v second=%+v", firstResult.Output, secondResult.Output)
	}
	records = client.Lifecycle.ListByTask("task-execute-1")
	if len(records) != 1 {
		t.Fatalf("idempotent retry executed again; lifecycle records=%d", len(records))
	}
}
