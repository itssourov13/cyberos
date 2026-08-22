# Phase 14 — Durable Autonomy & Handoff (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **14A** | Durable AutonomyRun repository | Done |
| **14B** | Orchestrator.Advance handoff | Done |
| **14C** | Operator API/SDK | Done |
| **14D** | Integration tests | Not started |
| **14E** | Freeze | Not started |

## 14A

AutonomyRunRepository (memory + PostgreSQL migration 014). Controller.Repo optional persist.
No Orchestrator.Advance. No Grant persistence.

## 14C Operator API

- `POST /api/v1/autonomy/runs`
- `GET /api/v1/autonomy/runs/{id}`
- `POST .../cancel` | `resume` | `advance`
SDK: StartAutonomy, GetAutonomy, CancelAutonomy, ResumeAutonomy, AdvanceAutonomy
Advance → AdvanceHandoff only (no direct Broker).
