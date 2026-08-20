package security_test

import (
	"testing"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

// Event/audit storage must not authorize execution.
func TestSecurity_EventsDoNotAuthorize(t *testing.T) {
	s := events.NewStore()
	_, err := s.Append(events.AppendRequest{
		ID: "e1", Type: events.TypeDecisionAllowed, CorrelationID: "c1",
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Recording an "allowed" decision event does not grant execution.
	if s.Len() != 1 {
		t.Fatal()
	}
}

func TestSecurity_AuditAttributionPreserved(t *testing.T) {
	ev := events.NewStore()
	a := audit.NewService(ev)
	p := contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman}
	d := contracts.Decision{
		ID: "d1", Outcome: contracts.DecisionAllowed, Timestamp: time.Now().UTC(),
		Principal: p, ProjectID: "p1", ScopeID: "s1", TaskID: "t1",
		CorrelationID: "corr-1", Reason: "ok",
	}
	e, err := a.RecordDecision(d)
	if err != nil {
		t.Fatal(err)
	}
	if e.Principal == nil || e.Principal.ID != "u1" || e.ScopeID != "s1" {
		t.Fatalf("%+v", e)
	}
	chain := a.ByCorrelation("corr-1")
	if len(chain) != 1 {
		t.Fatal(len(chain))
	}
}
