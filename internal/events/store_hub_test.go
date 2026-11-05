package events_test

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

func TestStorePublishesOnlyPersistedEvents(t *testing.T) {
	store := events.NewStore()
	hub := events.NewHub()
	store.WithHub(hub)
	ch := hub.Subscribe(1)
	defer hub.Unsubscribe(ch)

	req := events.AppendRequest{
		ID: "evt-1", Type: contracts.EventType("test.created"),
		CorrelationID: "corr-1", Timestamp: time.Now().UTC(),
		Payload: map[string]any{"ok": true},
	}
	if _, err := store.Append(req); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-ch:
		if got.ID != req.ID || got.CorrelationID != req.CorrelationID {
			t.Fatalf("published event mismatch: %+v", got)
		}
	default:
		t.Fatal("persisted event was not published to hub")
	}
}
