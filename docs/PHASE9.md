# Phase 9 — Tool Runtime & Pack Ecosystem (**FROZEN** at `v0.9.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **9A** | Process policy config + argv safety | Done |
| **9B** | Pack lifecycle / version / capability discovery | Done |
| **9C** | Constrained nmap + HTTP probe on Broker path | Done |
| **9D** | Durable execution records, idempotency, evidence completeness | Done |
| **9E** | Final integration audit | Done (read-only) |

## Security spine (unchanged)

```
Scope → Principal/Auth → Control Plane → Decision → Grant
  → Broker → Sandbox → Executor / ToolExecutor
  → Evidence + Provenance → Findings / Risk / Cases / Reports
```

Broker remains the sole product execution gateway. Pack registration and capability discovery never authorize execution.

## Process runtime

- Default: **fail-closed** (`CYBEROS_PROCESS_ENABLED=false`, no shell).
- Enable only with explicit allowlist (`CYBEROS_PROCESS_ALLOWED_BINS`, …).
- No user-controlled executable path; no `sh -c`.

## Known limitations (honest)

1. Live PostgreSQL durability is verified only when integration tests **PASS** with a reachable DSN; **SKIP ≠ verified**.
2. Process tools require operator allowlist configuration in production.
3. AI/JARVIS, Admin/Break-Glass, marketplace, and distributed workers remain out of scope.

## Constructors

| Constructor | Behavior |
|-------------|----------|
| `sdk.NewInMemory()` | Memory-only; lifecycle without PG repo |
| `sdk.NewProduction` | PostgreSQL + durable repos including execution records; fail-closed |
