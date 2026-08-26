# Phase 15 — Event-Driven Controlled Autonomy (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **15A** | Autonomy domain events + observation contracts | Done |
| **15B** | Event → Observe → Analyze → Re-plan | Not started |
| **15C** | Durable dispatcher / outbox | Not started |
| **15D** | Provider / external-brain hardening | Not started |
| **15E** | Freeze | Not started |

## 15A

Versioned contracts only:

- `AutonomyEvent` + `AutonomyEventContractV1`
- Types: `execution.completed|failed`, `finding.created`, `workflow.completed|failed`, `autonomy.policy_blocked`
- `EventObservationRequest` + `ObservationContractV1` (distinct from Phase 13 Analyze `ObservationRequest`)
- `ValidateAutonomyEvent` / `ToEventObservation` — pure, fail-closed
- Idempotency: `(project_id, idempotency_key)` via `SameLogicalEvent`
- `ToEnvelope()` for existing `EventEnvelope` / Hub compatibility
- Payload bounds; secret-like keys rejected

**STOP boundary:** Event → Observation → **STOP**

Does **not** invoke Brain, Policy, Handoff, Orchestrator, Grant, Broker, or Executor.

## Security

```
Event ≠ Authorization
Observation ≠ Execution
Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor
```
