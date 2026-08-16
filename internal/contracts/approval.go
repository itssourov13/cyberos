package contracts

// ApprovalStatus represents the state of an approval request.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "PENDING"
	ApprovalStatusApproved ApprovalStatus = "APPROVED"
	ApprovalStatusDenied   ApprovalStatus = "DENIED"
	ApprovalStatusExpired  ApprovalStatus = "EXPIRED"
	ApprovalStatusSkipped  ApprovalStatus = "SKIPPED" // policy did not require approval
)

// Valid reports whether the status is known.
func (s ApprovalStatus) Valid() bool {
	switch s {
	case ApprovalStatusPending, ApprovalStatusApproved,
		ApprovalStatusDenied, ApprovalStatusExpired, ApprovalStatusSkipped:
		return true
	default:
		return false
	}
}

// ApprovalResult captures the outcome of an approval check.
type ApprovalResult struct {
	ApprovalID ApprovalID     `json:"approval_id,omitempty"`
	Status     ApprovalStatus `json:"status"`
	Reason     string         `json:"reason,omitempty"`
	// Approver is the principal who approved/denied when applicable.
	Approver *PrincipalRef `json:"approver,omitempty"`
}
