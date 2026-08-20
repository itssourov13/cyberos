package contracts

// CaseStatus is the lifecycle of a security case.
type CaseStatus string

const (
	CaseStatusOpen         CaseStatus = "OPEN"
	CaseStatusAcknowledged CaseStatus = "ACKNOWLEDGED"
	CaseStatusInProgress   CaseStatus = "IN_PROGRESS"
	CaseStatusResolved     CaseStatus = "RESOLVED"
	CaseStatusAccepted     CaseStatus = "ACCEPTED"
	CaseStatusReopened     CaseStatus = "REOPENED"
)

// Valid reports whether the case status is known.
func (s CaseStatus) Valid() bool {
	switch s {
	case CaseStatusOpen, CaseStatusAcknowledged, CaseStatusInProgress,
		CaseStatusResolved, CaseStatusAccepted, CaseStatusReopened:
		return true
	default:
		return false
	}
}

// ReportFormat identifies an export format.
type ReportFormat string

const (
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatMarkdown ReportFormat = "markdown"
)

// Valid reports whether the report format is known.
func (f ReportFormat) Valid() bool {
	switch f {
	case ReportFormatJSON, ReportFormatMarkdown:
		return true
	default:
		return false
	}
}
