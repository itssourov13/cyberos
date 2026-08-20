package events

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestAppendOnlyAndQueries(t *testing.T) {
	s := NewStore()
	_, err := s.Append(AppendRequest{
		ID: "e1", Type: TypeDecisionAllowed, CorrelationID: "c1",
		Timestamp: time.Now().UTC(),
		Payload:   map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Fatal(s.Len())
	}
	got, err := s.Get("e1")
	if err != nil || got.CorrelationID != "c1" {
		t.Fatalf("%v %+v", err, got)
	}
	if len(s.ByCorrelation("c1")) != 1 {
		t.Fatal("correlation")
	}
	if len(s.ByType(TypeDecisionAllowed)) != 1 {
		t.Fatal("type")
	}
	_ = contracts.EventSchemaV1
}

func TestMemoryBackendSharedDurability(t *testing.T) {
	backend := NewMemoryBackend()
	s1 := NewStore().WithBackend(backend)
	_, _ = s1.Append(AppendRequest{
		ID: "e2", Type: TypeGrantIssued, CorrelationID: "c2", Timestamp: time.Now().UTC(),
	})
	s2 := NewStore().WithBackend(backend)
	if s2.Len() != 1 {
		t.Fatalf("len=%d", s2.Len())
	}
}
