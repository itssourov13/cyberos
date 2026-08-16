package evidence

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func sampleGrant() contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID:        "grant-1",
		Principal: contracts.PrincipalRef{ID: "user-1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-1", ScopeID: "scope-1", TaskID: "task-1", CapabilityID: "cap-1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(5 * time.Minute),
		CorrelationID: "corr-1",
	}
}

func TestRecordFromExecution(t *testing.T) {
	svc := NewService()
	res := contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
		Output:     map[string]any{"mock": true},
		ExecutorID: "mock",
	}
	ev, err := svc.RecordFromExecution(res, sampleGrant(), "ev-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureLinked(ev); err != nil {
		t.Fatal(err)
	}
	if ev.GrantID != "grant-1" || ev.TaskID != "task-1" || ev.Principal.ID != "user-1" {
		t.Errorf("linkage incomplete: %+v", ev)
	}
	if ev.Digest == "" {
		t.Error("expected digest")
	}
	got, err := svc.Get("ev-1")
	if err != nil || got.ID != "ev-1" {
		t.Fatalf("get: %v", err)
	}
}

func TestRecordRequiresGrantContext(t *testing.T) {
	svc := NewService()
	bad := sampleGrant()
	bad.TaskID = ""
	_, err := svc.RecordFromExecution(contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
	}, bad, "ev-x")
	if err == nil {
		t.Fatal("missing task on grant should fail")
	}
}
