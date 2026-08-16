package contracts

// FindingSeverity classifies the severity of a finding.
type FindingSeverity string

const (
	FindingSeverityInfo     FindingSeverity = "INFO"
	FindingSeverityLow      FindingSeverity = "LOW"
	FindingSeverityMedium   FindingSeverity = "MEDIUM"
	FindingSeverityHigh     FindingSeverity = "HIGH"
	FindingSeverityCritical FindingSeverity = "CRITICAL"
)

// Valid reports whether the severity is known.
func (s FindingSeverity) Valid() bool {
	switch s {
	case FindingSeverityInfo, FindingSeverityLow, FindingSeverityMedium,
		FindingSeverityHigh, FindingSeverityCritical:
		return true
	default:
		return false
	}
}

// FindingRef is a lightweight reference to a finding.
type FindingRef struct {
	ID FindingID `json:"id"`
}

// Validate checks required fields.
func (f FindingRef) Validate() error {
	return ValidateNonEmpty("finding.id", string(f.ID))
}

// FindingMetadata captures high-level attributes of a finding derived from evidence.
type FindingMetadata struct {
	ID            FindingID       `json:"id"`
	TaskID        TaskID          `json:"task_id"`
	EvidenceIDs   []EvidenceID    `json:"evidence_ids,omitempty"`
	Severity      FindingSeverity `json:"severity"`
	Title         string          `json:"title"`
	ProducedAt    Timestamp       `json:"produced_at"`
	CorrelationID CorrelationID   `json:"correlation_id,omitempty"`
}
