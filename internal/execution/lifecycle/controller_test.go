package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
)

func sampleGrant() contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID: "grant-1", Principal: contracts.PrincipalRef{ID: "user-1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj-1", ScopeID: "scope-1", TaskID: "task-1", CapabilityID: "cap-1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(5 * time.Minute),
		CorrelationID: "corr-life-1", Nonce: "nonce-life-1",
	}
}

func TestRunSuccessEnrichesEvidence(t *testing.T) {
	brk := broker.New(sandbox.New(mock.New()))
	ev := evidence.NewService()
	find := findings.NewService(ev)
	prov := provenance.NewService()
	ctrl := New(brk, WithEvidence(ev), WithFindings(find), WithProvenance(prov),
		WithResourcePolicy(contracts.DefaultResourcePolicy()))

	out, err := ctrl.Run(RunRequest{
		Grant: sampleGrant(), Parameters: map[string]any{"k": "v"},
		FindingTitle: "Observation", FindingSeverity: contracts.FindingSeverityInfo,
		CorrelationID: "corr-life-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Record.State != contracts.ExecutionLifecycleSucceeded {
		t.Fatalf("state=%s reason=%s", out.Record.State, out.Record.FailureReason)
	}
	if out.Record.GrantID != "grant-1" || out.Record.TaskID != "task-1" {
		t.Fatalf("correlation missing: %+v", out.Record)
	}
	if out.Evidence == nil {
		t.Fatal("expected evidence")
	}
	if out.Finding == nil {
		t.Fatal("expected finding")
	}
	// Provenance for evidence
	if _, err := prov.Get("evidence", string(*out.Evidence)); err != nil {
		t.Fatalf("provenance: %v", err)
	}
	// Evidence has lifecycle metadata
	got, _ := ev.Get(*out.Evidence)
	life, ok := got.Payload["lifecycle"].(map[string]any)
	if !ok {
		t.Fatalf("lifecycle metadata missing: %+v", got.Payload)
	}
	if life["execution_id"] == nil {
		t.Error("execution_id in lifecycle metadata")
	}
}

func TestRunNoGrantFails(t *testing.T) {
	ctrl := New(broker.New(mock.New()), WithEvidence(evidence.NewService()))
	_, err := ctrl.Run(RunRequest{})
	if err == nil {
		t.Fatal("empty grant must fail")
	}
}

func TestListByTask(t *testing.T) {
	ctrl := New(broker.New(sandbox.New(mock.New())), WithEvidence(evidence.NewService()))
	g := sampleGrant()
	g.Nonce = "n2"
	_, err := ctrl.Run(RunRequest{Grant: g, CorrelationID: "c2"})
	if err != nil {
		t.Fatal(err)
	}
	list := ctrl.ListByTask("task-1")
	if len(list) != 1 {
		t.Fatalf("list=%d", len(list))
	}
}

type memExecRepo struct {
	m      map[contracts.ExecutionID]contracts.ExecutionRecord
	byIdem map[string]contracts.ExecutionID
}

func (m *memExecRepo) Upsert(_ context.Context, r contracts.ExecutionRecord) error {
	if m.m == nil {
		m.m = map[contracts.ExecutionID]contracts.ExecutionRecord{}
		m.byIdem = map[string]contracts.ExecutionID{}
	}
	m.m[r.ID] = r
	if r.IdempotencyKey != "" {
		m.byIdem[r.IdempotencyKey] = r.ID
	}
	return nil
}
func (m *memExecRepo) GetByID(_ context.Context, id contracts.ExecutionID) (contracts.ExecutionRecord, error) {
	r, ok := m.m[id]
	if !ok {
		return contracts.ExecutionRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memExecRepo) GetByIdempotencyKey(_ context.Context, key string) (contracts.ExecutionRecord, error) {
	id, ok := m.byIdem[key]
	if !ok {
		return contracts.ExecutionRecord{}, contracts.ErrNotFound
	}
	return m.GetByID(context.Background(), id)
}
func (m *memExecRepo) ListByTask(_ context.Context, taskID contracts.TaskID, limit, offset int) ([]contracts.ExecutionRecord, error) {
	var out []contracts.ExecutionRecord
	for _, r := range m.m {
		if r.TaskID == taskID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *memExecRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]contracts.ExecutionRecord, error) {
	return nil, nil
}

func TestIdempotentRetryDoesNotDuplicate(t *testing.T) {
	repo := &memExecRepo{}
	ctrl := New(broker.New(sandbox.New(mock.New())),
		WithEvidence(evidence.NewService()),
		WithProvenance(provenance.NewService()),
		WithRepository(repo),
	)
	g := sampleGrant()
	g.Nonce = "idem-nonce-1"
	out1, err := ctrl.Run(RunRequest{Grant: g, IdempotencyKey: "idem-1", CorrelationID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	out2, err := ctrl.Run(RunRequest{Grant: g, IdempotencyKey: "idem-1", CorrelationID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if out1.Record.ID != out2.Record.ID {
		t.Fatalf("idempotent should return same record %s vs %s", out1.Record.ID, out2.Record.ID)
	}
	if out1.Evidence == nil || out2.Evidence == nil || *out1.Evidence != *out2.Evidence {
		t.Fatalf("idempotent retry must retain the evidence link: first=%v second=%v", out1.Evidence, out2.Evidence)
	}
	// Durable read-through after clearing memory
	ctrl.records = map[contracts.ExecutionID]*contracts.ExecutionRecord{}
	got, err := ctrl.Get(out1.Record.ID)
	if err != nil || got.ID != out1.Record.ID {
		t.Fatal(err, got)
	}
}

func TestStartedIdempotencyRecordBlocksRetry(t *testing.T) {
	repo := &memExecRepo{}
	g := sampleGrant()
	g.Nonce = "started-idempotency-nonce"
	if err := repo.Upsert(context.Background(), contracts.ExecutionRecord{
		ID: "execution-in-flight", State: contracts.ExecutionLifecycleStarted,
		TaskID: g.TaskID, GrantID: g.ID, Principal: g.Principal,
		ProjectID: g.ProjectID, ScopeID: g.ScopeID, CapabilityID: g.CapabilityID,
		Attempt: 1, StartedAt: time.Now().UTC(), IdempotencyKey: "workflow-1:step-1",
	}); err != nil {
		t.Fatal(err)
	}
	ctrl := New(broker.New(sandbox.New(mock.New())),
		WithEvidence(evidence.NewService()), WithRepository(repo))
	_, err := ctrl.Run(RunRequest{Grant: g, IdempotencyKey: "workflow-1:step-1"})
	if !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("uncertain STARTED execution must block retry, got %v", err)
	}
	if len(repo.m) != 1 {
		t.Fatalf("retry created another execution record: %d", len(repo.m))
	}
}

func TestEvidenceRequiredOnLifecycle(t *testing.T) {
	ctrl := New(broker.New(sandbox.New(mock.New()))) // no evidence
	_, err := ctrl.Run(RunRequest{Grant: sampleGrant()})
	if err == nil {
		t.Fatal("evidence service required for complete provenance path")
	}
}
