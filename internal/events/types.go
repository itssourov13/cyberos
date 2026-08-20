package events

import "cyberos/internal/contracts"

// Well-known event types for security-critical and runner lifecycle facts.
// Commands are NOT events; these are facts after the fact.
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

	// Task runner lifecycle (Phase 3C) — facts only; not authorization.
	TypeRunnerPromoted  contracts.EventType = "runner.task.promoted"
	TypeRunnerClaimed   contracts.EventType = "runner.task.claimed"
	TypeRunnerReleased  contracts.EventType = "runner.task.released"
	TypeRunnerRecovered contracts.EventType = "runner.lease.recovered"
)
