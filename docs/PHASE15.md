# Phase 15 — Event-Driven Controlled Autonomy (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **15A** | Autonomy domain events + observation contracts | Done |
| **15B** | Event → Observe → Analyze → Re-plan → Policy | Done |
| **15C** | Durable dispatcher / outbox | Done |
| **15D** | Provider / external-brain hardening | Done |
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

## 15B

`EventAutonomyProcessor.Process`:
Event → Observation → Analyze → RePlan → ValidatePlan → Apply → Policy → **STOP**

- MANUAL/ASSISTED → not AdvanceEligible
- CONTROLLED may report POLICY_ELIGIBLE without calling Handoff.Execute / Orchestrator
- In-process idempotency on (project, idempotency_key)
- Terminal/cancelled runs → TERMINAL; identity mismatch → STALE_CONTEXT

## 15C

Durable outbox (`015_autonomy_outbox.sql`) + memory/PG repos + `Dispatcher`.

- Claim/lease, bounded retry, dead-letter, per-run one-claim-per-batch ordering
- Delivers only to `EventAutonomyProcessor` (15B)
- No Grant/Broker/Executor path; live PG verification SKIP without DSN

## 15D

Provider hardening verified (timeout, LimitReader, redirect strips Authorization, APIKeyRef).
API: `POST /api/v1/ai/analyze|plan|apply` — project-scoped, untrusted results only.
SDK: `Analyze`, `Plan`, `ApplyPlan`, `WithBrain`.
External brain = authenticated principal; no Grant/Broker/Executor APIs.
