package lifecycle

import (
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
