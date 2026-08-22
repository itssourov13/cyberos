# CYBEROS Deployment

**Phase 0 Foundation Document**

## Current Status

Phase 0 produces a single binary. No deployment packaging yet.

## Future Directions

- Single-binary local deployment
- Container images
- Kubernetes / orchestrated deployment
- Multi-tenant cloud offering
- Distributed workers (only after modular monolith is proven)

## Configuration

- Environment variables
- Config files
- `.env` (development only; never commit secrets)

See `.env.example`.

## Phase 7C — Production vs development

### Development (default)

```bash
./cyberos serve
# or CLI without serve
./cyberos health
```

Uses `sdk.NewInMemory()`. Auth optional unless `CYBEROS_AUTH_REQUIRED=true`.

### Production

```bash
export CYBEROS_ENV=production
export CYBEROS_DATABASE_URL='postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable'
./cyberos serve
```

Behavior:

- `sdk.NewProduction` — PostgreSQL + migrations 001–010
- Durable findings, risk, cases, reports, API keys
- **Mandatory authentication** (no silent memory/unauth fallback)
- Startup **fail-closed** if DB or auth bootstrap fails
- Bootstrap API key printed once to stderr

Health/version remain public; other `/api/v1/*` routes require API key when auth is required.

Authentication never issues execution grants. Authorization remains Control Plane → Grant → Broker.


**Phase 7 frozen at tag `v0.7.0`.** See `docs/PHASE7.md` for limitations (including PG environment dependency).

## Phase 8D — Bootstrap, durable writes, PG verification

### Bootstrap API keys (production)

1. **Preferred:** set `CYBEROS_BOOTSTRAP_API_KEY` to an existing `cos_…` key (secret is **not** printed on startup).
2. If active keys already exist in the store, production **does not** issue a new bootstrap key.
3. Only when the key store is empty and no env key is set does serve create one bootstrap key and print it **once**.
4. Bootstrap/auth failure → process **exit 1** (fail-closed).

### Durable persistence

With `sdk.NewProduction`, findings/risk/evidence/provenance/cases/reports/assets inventory use repositories. Persist errors **propagate** (no silent `_ = Upsert` on findings/risk).

`NewInMemory()` remains memory-only for tests/dev.

### PostgreSQL integration tests

```bash
# No DSN: tests may SKIP (do not treat as durability verified)
go test -tags=integration ./...

# DSN set but DB down: strict tests FAIL
export CYBEROS_DATABASE_URL='postgres://…'
go test -tags=integration ./internal/storage/postgres/ -run StrictDSN
```

Never claim PG durability verified when tests were skipped.


**Phase 8 frozen at tag `v0.8.0`.** See `docs/PHASE8.md`.


**Phase 9 frozen at tag `v0.9.0`.** Process: `CYBEROS_PROCESS_ENABLED` (default false), allowlist bins. See `docs/PHASE9.md`.


**Phase 10 frozen at tag `v0.10.0`.** Workflow runs use migration 013; process tools still fail-closed by default.


**Phase 11 frozen at tag `v0.11.0`.** Load pack JSON via tools.Loader; process allowlist still operator-controlled.


**Phase 12 frozen at tag `v0.12.0`.** LLM providers optional; configure BaseURL + secret refs. Process tools remain fail-closed.
