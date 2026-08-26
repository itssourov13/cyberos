package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// EnqueueAutonomyEvent validates and persists an event to the outbox (no processing).
func EnqueueAutonomyEvent(ctx context.Context, repo ports.AutonomyOutboxRepository, e AutonomyEvent) error {
	if repo == nil {
		return fmt.Errorf("%w: outbox repository required", contracts.ErrInvalidArgument)
	}
	if err := ValidateAutonomyEvent(e); err != nil {
		return err
	}
	payload, err := json.Marshal(e.Payload)
	if err != nil || payload == nil {
		payload = []byte("{}")
	}
	rec := ports.AutonomyOutboxRecord{
		ID: e.EventID, ProjectID: e.ProjectID, TenantID: e.TenantID, ScopeID: e.ScopeID,
		AutonomyRunID: e.AutonomyRunID, WorkflowRunID: e.WorkflowRunID, ExecutionID: e.ExecutionID,
		EventType: string(e.Type), SchemaVersion: string(e.SchemaVersion),
		IdempotencyKey: e.IdempotencyKey, EventID: e.EventID, PayloadJSON: payload,
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(), CreatedAt: time.Now().UTC(),
	}
	return repo.Enqueue(ctx, rec)
}

// RecordToAutonomyEvent reconstructs AutonomyEvent from outbox record.
func RecordToAutonomyEvent(rec ports.AutonomyOutboxRecord) (AutonomyEvent, error) {
	var payload map[string]any
	if len(rec.PayloadJSON) > 0 {
		_ = json.Unmarshal(rec.PayloadJSON, &payload)
	}
	e := AutonomyEvent{
		EventID: rec.EventID, SchemaVersion: contracts.SchemaVersion(rec.SchemaVersion),
		Type: AutonomyEventType(rec.EventType), ProjectID: rec.ProjectID, TenantID: rec.TenantID,
		ScopeID: rec.ScopeID, AutonomyRunID: rec.AutonomyRunID, WorkflowRunID: rec.WorkflowRunID,
		ExecutionID: rec.ExecutionID, IdempotencyKey: rec.IdempotencyKey,
		Timestamp: rec.CreatedAt, Payload: payload,
	}
	if e.EventID == "" {
		e.EventID = rec.ID
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	return e, ValidateAutonomyEvent(e)
}
