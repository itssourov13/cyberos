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
