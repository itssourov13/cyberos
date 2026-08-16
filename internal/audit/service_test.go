package audit

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

func TestRecordDecisionAndGrant(t *testing.T) {
	es := events.NewStore()
	svc := NewService(es)

	d := contracts.Decision{
		ID: "dec-1", Outcome: contracts.DecisionAllowed, Reason: "ok",
		Timestamp: time.Now().UTC(),
		Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		CorrelationID: "corr-audit-1",
	}
	entry, err := svc.RecordDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Outcome != "ALLOWED" {
		t.Errorf("outcome = %s", entry.Outcome)
	}

	g := contracts.ExecutionGrant{
		ID: "g1", Principal: d.Principal,
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute),
		CorrelationID: "corr-audit-1", Nonce: "abc",
	}
	if _, err := svc.RecordGrantIssued(g); err != nil {
		t.Fatal(err)
	}

	if svc.Len() != 2 {
		t.Fatalf("audit len = %d", svc.Len())
	}
	if es.Len() != 2 {
		t.Fatalf("events len = %d", es.Len())
	}
	chain := svc.ByCorrelation("corr-audit-1")
	if len(chain) != 2 {
		t.Fatalf("correlation chain = %d", len(chain))
	}
}

func TestRecordExecution(t *testing.T) {
	svc := NewService(events.NewStore())
	g := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute),
		CorrelationID: "corr-2",
	}
	res := contracts.ExecutionResult{
		Status:    contracts.ExecutionStatusSucceeded,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
		ExecutorID: "mock",
	}
	e, err := svc.RecordExecution(g, res)
	if err != nil {
		t.Fatal(err)
	}
	if e.Outcome != "SUCCEEDED" {
		t.Errorf("outcome = %s", e.Outcome)
	}
}

func TestAuditRequiresCorrelation(t *testing.T) {
	svc := NewService(nil)
	d := contracts.Decision{
		ID: "d", Outcome: contracts.DecisionDenied, Reason: "no",
		Timestamp: time.Now().UTC(),
		Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
	}
	_, err := svc.RecordDecision(d)
	if err == nil {
		t.Fatal("missing correlation must fail")
	}
}
