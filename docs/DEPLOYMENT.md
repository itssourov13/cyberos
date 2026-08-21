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
