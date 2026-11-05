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
	TypePackTrusted        contracts.EventType = "trust.pack.trusted"
	TypePackRevoked        contracts.EventType = "trust.pack.revoked"
	TypeSignerRegistered   contracts.EventType = "trust.signer.registered"
	TypeSignerDisabled     contracts.EventType = "trust.signer.disabled"
	TypeSignerRevoked      contracts.EventType = "trust.signer.revoked"
	TypeGrantRejected      contracts.EventType = "grant.rejected"
	TypeGrantRevoked       contracts.EventType = "grant.revoked"
	TypePackBindRejected   contracts.EventType = "broker.pack_bind.rejected"

	// Task runner lifecycle (Phase 3C) — facts only; not authorization.
	TypeRunnerPromoted  contracts.EventType = "runner.task.promoted"
	TypeRunnerClaimed   contracts.EventType = "runner.task.claimed"
	TypeRunnerReleased  contracts.EventType = "runner.task.released"
	TypeRunnerRecovered contracts.EventType = "runner.lease.recovered"
)
