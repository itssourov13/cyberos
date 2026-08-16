package security_test

import (
	"testing"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

// Forensic: decision → grant chain share correlation ID.
func TestSecurity_AuditCorrelationChain(t *testing.T) {
	es := events.NewStore()
	a := audit.NewService(es)
	corr := contracts.CorrelationID("forensic-chain-1")

	d := contracts.Decision{
		ID: "dec-x", Outcome: contracts.DecisionAllowed, Reason: "ok",
		Timestamp: time.Now().UTC(),
		Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1",
		CorrelationID: corr,
	}
	if _, err := a.RecordDecision(d); err != nil {
		t.Fatal(err)
	}
	g := contracts.ExecutionGrant{
		ID: "g-x", Principal: d.Principal,
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Minute),
		CorrelationID: corr,
	}
	if _, err := a.RecordGrantIssued(g); err != nil {
		t.Fatal(err)
	}

	chain := a.ByCorrelation(corr)
	if len(chain) < 2 {
		t.Fatalf("expected forensic chain length >= 2, got %d", len(chain))
	}
	evts := es.ByCorrelation(corr)
	if len(evts) < 2 {
		t.Fatalf("expected events chain >= 2, got %d", len(evts))
	}
}

// Events are facts, not commands: envelope is immutable after append.
func TestSecurity_EventAppendOnly(t *testing.T) {
	s := events.NewStore()
	payload := map[string]any{"secret": "original"}
	_, err := s.Append(events.AppendRequest{
		ID: "e1", Type: events.TypeDecisionDenied,
		CorrelationID: "c1", Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload["secret"] = "tampered"
	if s.All()[0].Payload["secret"] != "original" {
		t.Fatal("event payload must not be mutable after append")
	}
}
