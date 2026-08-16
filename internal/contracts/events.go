package contracts

// EventType identifies the kind of event.
// Concrete event types will be defined as the system grows; the envelope
// remains stable.
type EventType string

// EventEnvelope is the versioned, immutable record of something that happened.
// Commands (requests to change state) and Events (facts) remain conceptually
// separate. This type is only for Events.
type EventEnvelope struct {
	ID            EventID       `json:"event_id"`
	Type          EventType     `json:"event_type"`
	SchemaVersion SchemaVersion `json:"schema_version"`
	Timestamp     Timestamp     `json:"timestamp"`

	// Actor and context
	Principal *PrincipalRef `json:"principal,omitempty"`
	ProjectID ProjectID     `json:"project_id,omitempty"`
	ScopeID   ScopeID       `json:"scope_id,omitempty"`
	TaskID    TaskID        `json:"task_id,omitempty"`

	// Causality
	CorrelationID CorrelationID `json:"correlation_id,omitempty"`
	CausationID   CausationID   `json:"causation_id,omitempty"`

	// Payload is the event-specific body. Prefer structured data.
	// SchemaVersion indicates how to interpret Payload.
	Payload map[string]any `json:"payload,omitempty"`
}

// Validate checks required envelope fields.
func (e EventEnvelope) Validate() error {
	if err := ValidateNonEmpty("event.id", string(e.ID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("event.type", string(e.Type)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("event.schema_version", string(e.SchemaVersion)); err != nil {
		return err
	}
	if e.Timestamp.IsZero() {
		return ErrInvalidArgument
	}
	return nil
}

// Common well-known schema version for Phase 1 events.
const EventSchemaV1 SchemaVersion = "1"
