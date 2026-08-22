# Phase 13 — Controlled Autonomous Cyber Operations (**FROZEN** at `v0.13.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **13A** | Observe → Analyze | Done |
| **13B** | Re-plan → Validate → Apply → Definition | Done |
| **13C** | Autonomy policy gate (MANUAL/ASSISTED/CONTROLLED) | Done |
| **13D** | Continuous AutonomyRun controller | Done |
| **13E** | Final audit + freeze | Done |

## Architecture

```
Human/API/Future JARVIS
        ↓
   AI Brain (untrusted Analyze/Plan)
        ↓
   Plan IR / AnalysisResult
        ↓
   Autonomy Policy (eligibility only)
        ↓
   Workflow Orchestrator
        ↓
   CP → Decision → Grant → Broker → Sandbox → Executor
        ↓
   Evidence → AIContext → Re-plan (bounded)
```

**13D boundary:** `Controller.Advance` returns `MayAdvance`; it does **not** call Orchestrator, GrantIssuer, or Broker. Callers must invoke the existing Orchestrator for actual step execution.

## Modes

| Mode | Behavior |
|------|----------|
| MANUAL | Plan/Definition only; WAIT — no auto-advance |
| ASSISTED | Same — WAIT for operator |
| CONTROLLED_AUTONOMOUS | ALLOW only if policy budgets/caps pass |

## Security kernel (unchanged)

Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor

AI ≠ authorization. Policy ≠ Grant. Discovery ≠ authorization.

## Limitations

1. AutonomyRun persistence is **in-memory** (process-local).
2. Live PostgreSQL autonomy durability **not verified**.
3. External JARVIS/Zoya product, remote workers, marketplace — **future**.
4. Full E2E auto-loop wiring Orchestrator inside Controller — intentional boundary (eligibility only).
