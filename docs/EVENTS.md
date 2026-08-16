# CYBEROS Event Architecture

**Phase 0 Foundation Document**

## Command vs Event

Strict separation:

- **Command** — intent to change state (e.g. `CreateTask`)
- **Event** — immutable fact that something occurred (e.g. `TaskCreated`)

## Event Envelope (Required Fields)

Every event supports at minimum:

- `event_id`
- `event_type`
- `schema_version`
- `timestamp`
- `principal`
- `project_id`
- `scope_id`
- `task_id`
- `correlation_id`
- `causation_id`
- `payload`

## Initial Implementation

PostgreSQL-backed event store is sufficient for early phases.

Prepare for future:

- event streaming
- WebSocket
- SIEM integration
- ZOYA observation
- audit pipelines
- automation triggers

Do **not** introduce Kafka or similar infrastructure in Phase 0/1.

## Phase 1I — Implemented

### Event Store (`internal/events.Store`)
- Versioned EventEnvelope (schema v1)
- Required: event_id, type, schema_version, timestamp, **correlation_id**
- Optional: principal, project/scope/task, causation_id, payload
- Append-only; payload copied on append (immutability)
- Query: All, ByCorrelation, ByType

### Well-known types
- controlplane.decision.allowed / denied
- grant.issued
- execution.succeeded / failed / rejected
- evidence.recorded, finding.created
- task.created, task.state_changed

### Audit (`internal/audit.Service`)
- Append-only audit entries for decisions, grants, executions, evidence, findings, task state changes
- Emits corresponding Events when an event store is wired
- ByCorrelation for forensic reconstruction

Commands remain separate from Events (this package only records facts).
