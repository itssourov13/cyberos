package events

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestHubPublishSubscribe(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe(4)
	env := contracts.EventEnvelope{
		ID: "e1", Type: TypeRunnerClaimed, SchemaVersion: contracts.EventSchemaV1,
		Timestamp: time.Now().UTC(), CorrelationID: "c1",
	}
	h.Publish(env)
	select {
	case got := <-ch:
		if got.ID != "e1" {
			t.Fatalf("got %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	h.Unsubscribe(ch)
	if h.Len() != 0 {
		t.Fatal("expected no subscribers")
	}
}
