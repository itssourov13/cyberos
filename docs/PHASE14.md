# Phase 14 — Durable Autonomy & Controlled Handoff (**FROZEN** at `v0.14.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **14A** | Durable AutonomyRun repository | Done |
| **14B** | Orchestrator.Advance handoff | Done |
| **14C** | Operator API/SDK | Done |
| **14D** | Lineage/budget/reconciliation security audit | Done |
| **14E** | Final audit + freeze | Done |

## Architecture

```
Human / External Brain (JARVIS/Zoya/LLM)
        │ API/SDK + Auth
        ▼
Autonomy Controller (durable run, lineage, budgets, cancel)
        │
Autonomy Policy (eligibility only — not a Grant)
        │ MayAdvance
        ▼
Advance Handoff
        │
Workflow Orchestrator
        │
CP → Decision → Grant → Broker → Sandbox → Executor
        │
Evidence / Findings → AI Context → Re-plan
```

## Trust boundaries

| Layer | Role |
|-------|------|
| Brain / Plan / Analysis | **Untrusted** proposal |
| Autonomy Policy | **Eligibility** gate only |
| Control Plane | **Authorization** authority |
| Grant | **Ephemeral** per-step artifact |
| Broker | **Sole** execution gateway |
| Orchestrator | Workflow execution authority |

## Modes

| Mode | Auto-advance |
|------|----------------|
| MANUAL | WAIT — no Orchestrator |
| ASSISTED | WAIT — no Orchestrator |
| CONTROLLED_AUTONOMOUS | ALLOW only if policy budgets/caps pass → Handoff → Orchestrator |

## Durability

- **AutonomyRun**: memory + PostgreSQL (`014_autonomy_runs.sql`); policy snapshot, steps_consumed, cancel, version CAS
- **WorkflowRun**: authoritative for step completion / no re-exec on restart
- **Grant**: never persisted in autonomy state
- Live PG verification: **SKIP** unless DSN configured (not claimed PASS)

## API / SDK

- `POST/GET /api/v1/autonomy/runs`, `.../cancel`, `.../resume`, `.../advance`
- SDK: `StartAutonomy`, `GetAutonomy`, `CancelAutonomy`, `ResumeAutonomy`, `AdvanceAutonomy`
- Advance → `AdvanceHandoff` only (no direct Broker/GrantIssuer)

## Limitations

1. Live PostgreSQL autonomy durability not verified without DSN.
2. External JARVIS/Zoya product, event bus, remote workers — future.
3. Full E2E Orchestrator materialization may require packs/lifecycle wiring in deployment.
4. Autonomy continuous re-plan loop remains operator/API driven (no background daemon).

## Security kernel (unchanged)

```
Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor
```
