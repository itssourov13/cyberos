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

// FindingStatus is the lifecycle of an observed finding.
type FindingStatus string

const (
	FindingStatusNew       FindingStatus = "NEW"
	FindingStatusUnchanged FindingStatus = "UNCHANGED"
	FindingStatusUpdated   FindingStatus = "UPDATED"
	FindingStatusReopened  FindingStatus = "REOPENED"
	FindingStatusResolved  FindingStatus = "RESOLVED"
)

// Valid reports whether the finding status is known.
func (s FindingStatus) Valid() bool {
	switch s {
	case FindingStatusNew, FindingStatusUnchanged, FindingStatusUpdated,
		FindingStatusReopened, FindingStatusResolved:
		return true
	default:
		return false
	}
}

// FindingConfidence is a simple confidence band (not a scoring engine).
type FindingConfidence string

const (
	FindingConfidenceLow    FindingConfidence = "LOW"
	FindingConfidenceMedium FindingConfidence = "MEDIUM"
	FindingConfidenceHigh   FindingConfidence = "HIGH"
)

// Valid reports whether confidence is known.
func (c FindingConfidence) Valid() bool {
	switch c {
	case FindingConfidenceLow, FindingConfidenceMedium, FindingConfidenceHigh:
		return true
	default:
		return false
	}
}
