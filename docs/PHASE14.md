# Phase 14 — Durable Autonomy & Handoff (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **14A** | Durable AutonomyRun repository | Done |
| **14B** | Orchestrator.Advance handoff | Done |
| **14C** | Operator API/SDK | Done |
| **14D** | Lineage/budget/reconciliation security audit | Done |
| **14E** | Freeze | Not started |

## 14A

AutonomyRunRepository (memory + PostgreSQL migration 014). Controller.Repo optional persist.
No Orchestrator.Advance. No Grant persistence.

## 14B

AdvanceHandoff: Policy ALLOW + CONTROLLED_AUTONOMOUS → Orchestrator.Advance.
MANUAL/ASSISTED → WAIT. Fresh Grant remains inside Orchestrator.

## 14C Operator API

- `POST /api/v1/autonomy/runs`
- `GET /api/v1/autonomy/runs/{id}`
- `POST .../cancel` | `resume` | `advance`
SDK: StartAutonomy, GetAutonomy, CancelAutonomy, ResumeAutonomy, AdvanceAutonomy
Advance → AdvanceHandoff only (no direct Broker).

## 14D Security audit

Focused tests cover: lineage, budget persistence across restart, MANUAL/ASSISTED
no-orchestrator, cancellation/terminal protection, project/tenant/scope isolation,
no Grant fields in AutonomyRun JSON, start idempotency, concurrent advance safety,
unknown policy version fail-closed, API cancel/advance regression.

Live PG autonomy verification: SKIP unless DSN configured (not claimed PASS).
