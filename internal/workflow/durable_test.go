package workflow_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
	"cyberos/internal/workflow"
)

type memRunRepo struct {
	mu   sync.Mutex
	data map[string]ports.WorkflowRunRecord
	fail bool
}

func (m *memRunRepo) Upsert(_ context.Context, r ports.WorkflowRunRecord) error {
	if m.fail {
		return errors.New("persist failed")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = map[string]ports.WorkflowRunRecord{}
	}
	m.data[r.ID] = r
	return nil
}
func (m *memRunRepo) GetByID(_ context.Context, id string) (ports.WorkflowRunRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.data[id]
	if !ok {
		return ports.WorkflowRunRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memRunRepo) ListByProject(_ context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.WorkflowRunRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ports.WorkflowRunRecord
	for _, r := range m.data {
		if r.ProjectID == projectID {
			out = append(out, r)
		}
	}
	return out, nil
}

func TestDurableRestartReadBack(t *testing.T) {
	repo := &memRunRepo{}
	orch := setupOrch(t, grantAuth(t)).WithRepository(repo)
	def := sampleDef()
	run, err := orch.Start(context.Background(), workflow.StartRequest{
		Definition: def, PrincipalID: "u1", RunID: "run-dur-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// clear memory
	orch2 := setupOrch(t, grantAuth(t)).WithRepository(repo)
	got, err := orch2.Get("run-dur-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != run.ID || got.ProjectID != "proj-a" {
		t.Fatal(got)
	}
}

func TestPersistFailureFailClosed(t *testing.T) {
	repo := &memRunRepo{fail: true}
	orch := setupOrch(t, grantAuth(t)).WithRepository(repo)
	_, err := orch.Start(context.Background(), workflow.StartRequest{
		Definition: sampleDef(), PrincipalID: "u1", RunID: "run-fail",
	})
	if err == nil {
		t.Fatal("persist failure must fail closed")
	}
}

func TestListByProjectIsolation(t *testing.T) {
	repo := &memRunRepo{}
	orch := setupOrch(t, grantAuth(t)).WithRepository(repo)
	_, err := orch.Start(context.Background(), workflow.StartRequest{
		Definition: sampleDef(), PrincipalID: "u1", RunID: "run-iso",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := orch.ListByProject("proj-a", 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}
	list, err = orch.ListByProject("other", 10, 0)
	if err != nil || len(list) != 0 {
		t.Fatal(err, list)
	}
}

func TestCompletedStepNotReExecuted(t *testing.T) {
	var grants int
	auth := workflow.FuncAuthorizer{Fn: func(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
		grants++
		return grantAuth(t).AuthorizeStep(ctx, req)
	}}
	orch := setupOrch(t, auth).WithRepository(&memRunRepo{})
	def := workflow.Definition{
		ID: "wf-once", Name: "O", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "only", CapabilityID: "cap.scan"}},
	}
	run, err := orch.Start(context.Background(), workflow.StartRequest{Definition: def, PrincipalID: "u1", RunID: "once"})
	if err != nil {
		t.Fatal(err)
	}
	run, err = orch.Advance(context.Background(), run.ID, def)
	if err != nil {
		t.Fatal(err)
	}
	g1 := grants
	run, err = orch.Advance(context.Background(), run.ID, def)
	if err != nil {
		t.Fatal(err)
	}
	if grants != g1 {
		t.Fatalf("completed step re-authorized: %d -> %d", g1, grants)
	}
	if run.State != workflow.RunStateSucceeded {
		t.Fatal(run.State)
	}
}
