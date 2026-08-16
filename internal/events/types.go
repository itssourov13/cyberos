package events

import "cyberos/internal/contracts"

// Well-known event types for Phase 1 security-critical actions.
// Commands (e.g. CreateTask) are NOT events; these are facts after the fact.
const (
	TypeDecisionAllowed    contracts.EventType = "controlplane.decision.allowed"
	TypeDecisionDenied     contracts.EventType = "controlplane.decision.denied"
	TypeGrantIssued        contracts.EventType = "grant.issued"
	TypeExecutionSucceeded contracts.EventType = "execution.succeeded"
	TypeExecutionFailed    contracts.EventType = "execution.failed"
	TypeExecutionRejected  contracts.EventType = "execution.rejected"
	TypeEvidenceRecorded   contracts.EventType = "evidence.recorded"
	TypeFindingCreated     contracts.EventType = "finding.created"
	TypeTaskCreated        contracts.EventType = "task.created"
	TypeTaskStateChanged   contracts.EventType = "task.state_changed"
	TypeScopeCreated       contracts.EventType = "scope.created"
	TypePrincipalCreated   contracts.EventType = "principal.created"
)
