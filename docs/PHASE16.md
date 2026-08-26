# Phase 16 — Production Durability & AI/Autonomy Guardrails

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **16A** | Production persistence & crash recovery | **Done** |
| **16B** | Outbox concurrency (SKIP LOCKED) | **Done** |
| **16C** | AI/autonomy LLM budgets | **Done** |
| **16D** | API rate limits / operator safety | **Done** |
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


## 16B

### Implemented
- PostgreSQL `ClaimBatch` uses a transaction + `FOR UPDATE SKIP LOCKED` CTE (atomic claim).
- Concurrent dispatchers cannot both own the same outbox row.
- Per-run ordering: at most one event per `autonomy_run_id` per batch.
- Lease recovery / active-lease protection preserved; attempts increment on claim.
- Integration tests: concurrent claim, active lease, expired reclaim (`-tags=integration`).

### Security
Outbox claim is **ownership only** — not authorization. No Grant persistence or Broker calls.

### Verification
- Memory tests: PASS
- Race autonomy/memory: run with suite
- Live PG: SKIP when `CYBEROS_DATABASE_URL` unset


## 16C

### Guardrails
- `MaxLLMCalls` — CONTROLLED requires >0; exhaustion → POLICY_BLOCKED
- `Cooldown` — `NextAllowedAt` persisted; blocks ReserveLLMCall
- `MaxNoProgress` — fingerprint via plan/workflow definition id; threshold blocks
- `ReserveLLMCall` before Brain; concurrent reservation safe via sequential store + versioned persist

### Persistence
Migration `016_autonomy_guardrails.sql`: `llm_calls_used`, `no_progress_count`, `last_progress_key`, `next_allowed_at`

### Security
Guardrail block → no Handoff/Orchestrator/Grant/Broker. MANUAL/ASSISTED remain WAIT for auto-advance.


## 16D

### API safety
- `http.MaxBytesReader` body cap (default 1 MiB)
- Process-local fixed-window rate limit on `/api/v1/ai/*` and `/api/v1/autonomy/*`
- Key: `(group|principal_id|project)`; HTTP **429** + `Retry-After`
- Rate limit does **not** consume 16C LLM budget

### ASSISTED approval
- `POST .../approve` and `.../reject` — control signals only
- ASSISTED starts `PENDING_APPROVAL`; Advance waits until `APPROVED`
- Approval ≠ Grant; Policy still WAIT for ASSISTED; cancel wins
- Migration `017_autonomy_approval.sql`

### Limitations
- Rate limiter is **process-local** (not global multi-replica)
- LIVE PG verification depends on DSN
