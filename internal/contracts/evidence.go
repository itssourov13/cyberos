package contracts

// EvidenceRef is a lightweight reference to evidence produced by an execution.
type EvidenceRef struct {
	ID EvidenceID `json:"id"`
}

// Validate checks required fields.
func (e EvidenceRef) Validate() error {
	return ValidateNonEmpty("evidence.id", string(e.ID))
}

// EvidenceMetadata captures high-level attributes of evidence.
// Full content storage is the responsibility of later layers.
type EvidenceMetadata struct {
	ID           EvidenceID       `json:"id"`
	TaskID       TaskID           `json:"task_id"`
	GrantID      ExecutionGrantID `json:"grant_id"`
	CapabilityID CapabilityID     `json:"capability_id"`
	Principal    PrincipalRef     `json:"principal"`
	ProducedAt   Timestamp        `json:"produced_at"`
	// ContentType / MediaType of the evidence payload (e.g. "application/json").
	ContentType string `json:"content_type,omitempty"`
	// Digest is a content hash for integrity (algorithm + value).
	Digest        string        `json:"digest,omitempty"`
	CorrelationID CorrelationID `json:"correlation_id,omitempty"`
}
