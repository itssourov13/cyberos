# Phase 10 — Workflow / Assessment Orchestration (**FROZEN** at `v0.10.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **10A** | Capability-driven workflow definition + materializer | Done |
| **10B** | Orchestrator + per-step CP/Grant + Lifecycle | Done |
| **10C** | Durable workflow_runs + restart-safe state | Done |
| **10D** | Production/SDK/API wiring | Done |
| **10E** | Final audit + freeze | Done |

## Execution path (sole gateway)

```
Workflow Definition → Validate → Capability Resolution → Task Engine
  → Orchestrator (ready steps only)
  → fresh CP.Evaluate → Decision → Grant
  → Lifecycle.Run → Broker → Sandbox → Executor
  → ExecutionRecord → Evidence / Provenance → Findings / Risk / …
```

Workflow, Task Engine, CapabilityResolver, and Registry are **orchestration/discovery only**. They never authorize or execute tools.

## PostgreSQL

Migration `013_workflow_runs` + `WorkflowRunRepository`.  
Live PG integration in this freeze environment: **not verified** (DSN absent → tests SKIP; SKIP ≠ verification).

## Known limitations

1. Live multi-host PG durability for workflow_runs must be proven with a real DSN before production claims.
2. Operator must seed identity/scope/capabilities for full CP-backed runs.
3. Process tools remain fail-closed until allowlisted (`CYBEROS_PROCESS_*`).
4. AI/JARVIS, Admin, marketplace, remote workers, UI redesign — out of scope.
