# Phase 6 Freeze (6H)

**Tag:** `v0.6.0` (Phase 6H production closure)

## What 6H closed

1. **E2E product path** after Broker execution:
   Assessment step success → Asset/Observation → Finding ingest → Risk prioritization → Case → Report export (JSON/MD)

2. **Pipeline:** `assessment.Phase6Sink` + `Orchestrator.WithPhase6` (wired from `sdk.WireAssessment`)

3. **Auth posture:** `CYBEROS_AUTH_REQUIRED` / `CYBEROS_ENV=production` enables API middleware in `cmd/cyberos serve`

4. **Security:** Phase 6 sinks never authorize; Broker remains sole execution gateway

## Still default in-memory

Findings, assets inventory, riskprio, cases, reports, API keys use in-process stores by default.
Migrations `006`–`010` remain available for future durable adapters — schemas exist; full PG wiring of every Phase 6 store is deferred beyond freeze (catalog/tasks/events already have adapters).

## Production notes

- Set `CYBEROS_AUTH_REQUIRED=true` (or `CYBEROS_ENV=production`)
- Bootstrap API key is printed once at serve start when auth required
- Process execution remains fail-closed unless policy allowlist is enabled

## Phase 7A progress (post-freeze)

Findings and risk prioritization evaluations now have **optional** PostgreSQL repository ports/adapters (`ports.FindingRepository`, `ports.RiskEvaluationRepository`, `postgres.FindingRepo`, `postgres.RiskEvalRepo`).

- `findings.Service.WithRepository` / `riskprio.Engine.WithRepository` dual-write and read-through
- `NewInMemory()` remains in-memory only
- Cases, reports, API keys, and production main wiring remain Phase 7B/7C
