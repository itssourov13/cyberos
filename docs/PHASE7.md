# Phase 7 — Durable Platform Completion (**FROZEN** at `v0.7.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **7A** | Findings + risk evaluation PostgreSQL ports/adapters | Done |
| **7B** | Cases, reports, API keys PostgreSQL | Done |
| **7C** | `sdk.NewProduction`, mandatory auth on production serve | Done |
| **7D** | Assessment → Report durable E2E + restart simulation | Done |
| **7E** | Final audit, docs freeze, release tag | Done |

## Security spine (unchanged)

```
Scope → Principal → Auth → Control Plane → Decision → Grant
  → Broker → Sandbox → Executor → Phase6Sink
  → Findings / Risk / Cases / Reports
```

- **Broker** is the sole product execution gateway.
- **No valid grant ⇒ no execution.**
- Auth does **not** issue grants.
- Production does **not** silently fall back to unauthenticated memory mode.

## Constructors

| Constructor | Use |
|-------------|-----|
| `sdk.NewInMemory()` | Tests, local CLI, development serve |
| `sdk.NewProduction(ctx, cfg, migrationsDir)` | `CYBEROS_ENV=production` + `CYBEROS_DATABASE_URL` |

## Known limitations

1. **Live PostgreSQL** integration tests skip when no server is reachable on the configured DSN. Restart durability against a real DB is **code-complete** but **environment-dependent**.
2. Asset **inventory** (observations/endpoints) remains primarily in-process; domain `assets` table has a repo for core asset rows.
3. Process execution remains **fail-closed** by default (`process.DefaultPolicy.Enabled=false`).
4. Admin / Break-Glass / AI / ZOYA / JARVIS / UAB / distributed workers are **out of scope** for Phase 7.

## Validation (7E)

- `go test ./...` — required green
- `go vet ./...` / `go build ./...` — required green
- `go test -tags=integration ./...` — green with PG tests **SKIP** if DB unavailable
