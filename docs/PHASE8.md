# Phase 8 — Operational Hardening & Multi-Tenant Readiness (**FROZEN** at `v0.8.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **8A** | Durable Evidence + Provenance (migration 011) | Done |
| **8B** | Durable asset endpoints/observations (migration 008) | Done |
| **8C** | API tenant + project isolation on reads | Done |
| **8D** | Bootstrap hygiene, strict findings/risk persist, PG harness | Done |

## Security spine (unchanged)

```
Scope → Principal/Auth → Control Plane → Decision → Grant
  → Broker → Sandbox → Executor → Phase6Sink → product data
```

Broker remains the sole product execution gateway. Auth ≠ authorization.

## Constructors

| Constructor | Behavior |
|-------------|----------|
| `sdk.NewInMemory()` | Memory-only; tests/dev/CLI default |
| `sdk.NewProduction` | PostgreSQL + durable repos; fail-closed |

## Known limitations (honest)

1. **Live PostgreSQL** durability tests **SKIP** when no DB is reachable; they are **not** claimed verified in environments without Postgres.
2. When `CYBEROS_DATABASE_URL` is set and DB is down, strict tests **FAIL**.
3. Findings/assets without `TenantID` rely on **project scope** for isolation.
4. Process execution remains fail-closed by default.
5. AI/Admin/Break-Glass/new tool families remain out of scope.
