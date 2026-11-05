# CYBEROS

**Cybersecurity Operating System** — Go-first modular monolith and control plane for authorized security work.

This is **not** only a scanner, dashboard, or chatbot. It is an auditable execution and authorization platform for human operators and (optional) automation.

## Current status (code-verified)

| Layer | Status |
|-------|--------|
| Security spine (CP → Grant → Broker → Host/Worker) | **CONFIRMED** (Go; unit tests) |
| Pack trust / digest binding / TrustAdmin | **CONFIRMED** |
| Remote mTLS + assignment + signed results | **CONFIRMED** (unit); live fleet **UNVERIFIED** without env |
| API / CLI / SDK | **CONFIRMED** |
| PostgreSQL durability | **PARTIAL** (migrations + adapters); live **UNVERIFIED** without `DATABASE_URL` |
| Web UI (`frontend/`) | **Phase 24C implemented; audit fixes applied; re-verification pending** — clean-slate React/TS/Vite 8 site + console on preview data; **not** API-integrated (Phase 24D pending owner acceptance). See `docs/FRONTEND_REBUILD.md` |
| Git HEAD (committed) | Phase **23C** docs/harness (`be855b9` family). `docs/AUDIT_FINAL_24B.md` is **historical** (pre-24C); the latest state is the post-24C audit plus the fix pass recorded in `state/HANDOFF.md` |
| API listen address | Dev default `127.0.0.1:8080`. A non-loopback `CYBEROS_API_ADDR` requires `CYBEROS_AUTH_REQUIRED=true` (startup is refused otherwise). Timeouts: `CYBEROS_API_{READ_HEADER,READ,WRITE,IDLE}_TIMEOUT` |

Highest tag seen: **v0.15.0**. Frontend may exist as uncommitted worktree content.

## Security spine (authoritative)

```
Scope → Auth → Control Plane → Decision → Grant → Broker → Host → Sandbox → Executor
         → Evidence → Provenance → Event → Audit
```

**Remote:**

```
Broker → RemoteHost → mTLS Worker → Assignment → Signed Result → Verify → Audit
```

Invariants:

- **Broker is the sole execution gateway**
- **TRUSTED ≠ AUTHORIZED** (pack trust is not a grant)
- **CLI / API / UI are not authorization kernels**
- **AI / Event must not mint Grants or call Broker**

## Quick start

```bash
make build
./bin/cyberos version
# Operator CLI surfaces: status, doctor, evaluate, grant, execute, trust-help, workers, …
```

Production SDK wiring requires DSN; missing DSN **fail-closed** (`NewProduction`). Tests may use `NewInMemory()`.

## Repository layout

```
cmd/                 # binaries
internal/            # Go implementation (private)
  controlplane/      # Decision authority
  broker/            # Grant issuer + sole Execute gateway
  execution/         # Host, sandbox, mTLS, assignment, resultauth
  api/v1/            # HTTP API
  cli/               # CLI
  sdk/               # NewInMemory / NewProduction
  ai/                # Optional; isolated from Broker
migrations/          # PostgreSQL versioned schema
frontend/            # React site + operator console (preview adapter; thin, non-authoritative client)
docs/                # Architecture, security, phase notes
tests/               # Integration harnesses (skip without env)
```

## Documentation map

| Doc | Purpose |
|-----|---------|
| [docs/AUDIT_FINAL_24B.md](docs/AUDIT_FINAL_24B.md) | **Latest full audit** |
| [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md) | Security invariants |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture |
| [docs/TRUST.md](docs/TRUST.md) | Pack trust / signers |
| [docs/WORKERS.md](docs/WORKERS.md) | Remote workers / mTLS |
| [docs/CLI.md](docs/CLI.md) | CLI operator surfaces |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Phase status |
| [docs/API.md](docs/API.md) | API boundary |
| [frontend/README.md](frontend/README.md) | UI thin-client notes |
| [docs/FRONTEND_REBUILD.md](docs/FRONTEND_REBUILD.md) | Phase 24C frontend rebuild |

## What must not be bypassed

Do not add parallel execution paths, client-side Grant minting, AI→Broker shortcuts, or silent Production fallbacks to in-memory when durability is required.

## License

See [LICENSE](LICENSE).
