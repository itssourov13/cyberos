# Phase 16 — Production Durability & AI/Autonomy Guardrails

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **16A** | Production persistence & crash recovery | **Done** |
| **16B** | Outbox concurrency (SKIP LOCKED) | Not started |
| **16C** | AI/autonomy LLM budgets | Not started |
| **16D** | API rate limits / operator safety | Not started |
| **16E** | Final audit + freeze | Not started |

## 16A

### Implemented
- AutonomyRun version CAS kept in sync after durable Update (restart-safe successive updates).
- `WorkflowRunID` / `CancelledAt` mapped through persist/LoadFromRepo.
- Restart tests: budget `steps_consumed` survives; cancel survives and blocks MayAdvance; project isolation on load.
- Outbox lease recovery test (memory): claim → crash → lease expiry → reclaim.
- PG integration tests under `-tags=integration` when `CYBEROS_DATABASE_URL` is set.

### Persistence model
`AutonomyRunRepository` (memory + PostgreSQL migration 014) stores policy snapshot, state, steps_consumed, lineage IDs, cancellation — **never** Grant or secrets.

### Workflow reconciliation
AutonomyRun holds `workflow_run_id` linkage only. Workflow step completion remains authoritative in the workflow layer; recovery does not re-execute completed steps.

### PostgreSQL verification
- Adapters: present for run + outbox (015).
- **LIVE PG:** SKIP when DSN unset; integration tests fail-closed if DSN set but unreachable.

### Limitations
1. Live PG not exercised without DSN in default CI.
2. Outbox claim still select+update (16B).
3. Dual-write (execution evidence ↔ outbox) not global-TX.
4. No LLM-call budget yet (16C).

### Security
AI / Event / Policy / Handoff do not mint Grants or call Broker.
