package events

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestAppendAndQuery(t *testing.T) {
	s := NewStore()
	p := contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman}
	env, err := s.Append(AppendRequest{
		ID: "e1", Type: TypeDecisionAllowed,
		Principal: &p, ProjectID: "p1", ScopeID: "s1", TaskID: "t1",
		CorrelationID: "corr-1", CausationID: "cause-1",
		Payload: map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if env.SchemaVersion != contracts.EventSchemaV1 {
		t.Error("schema version")
	}
	if s.Len() != 1 {
		t.Fatalf("len = %d", s.Len())
	}
	chain := s.ByCorrelation("corr-1")
	if len(chain) != 1 {
		t.Fatalf("correlation chain = %d", len(chain))
	}
	if len(s.ByType(TypeDecisionAllowed)) != 1 {
		t.Error("by type")
	}
}

func TestAppendRequiresCorrelation(t *testing.T) {
	s := NewStore()
	_, err := s.Append(AppendRequest{
		ID: "e2", Type: TypeGrantIssued,
	})
	if err == nil {
		t.Fatal("correlation_id required")
	}
}

func TestAppendOnlyImmutableSnapshot(t *testing.T) {
	s := NewStore()
	payload := map[string]any{"x": 1}
	_, err := s.Append(AppendRequest{
		ID: "e3", Type: TypeTaskCreated,
		CorrelationID: "c", Payload: payload,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload["x"] = 999 // mutate original
	all := s.All()
	if all[0].Payload["x"] != 1 {
		t.Error("store must copy payload (immutability)")
	}
}
