# CYBEROS Roadmap

Status labels: **DONE** · **IMPLEMENTED / EXTERNAL CHECKS PENDING** · **PARTIAL** · **DEFERRED**

| Phase | Topic | Status |
|---|---|---|
| 1–7 | Foundation → platform freeze | **DONE** |
| 8–9 | Durability, tools, process policy | **DONE** |
| 10–17 | Workflows, autonomy isolation, hardening | **DONE** (see phase docs) |
| 18 | Pack integrity / signature | **DONE** |
| 19 | Grant pack binding, durable trust, audit | **DONE** |
| 20 | Atomic durable grant claim | **DONE** (local tests; live multi-node check pending) |
| 21 | TrustAdmin | **DONE** |
| 22 | mTLS, result auth, revoke, regression | **DONE** (unit tests; live fleet check pending) |
| 23A–23C | Security/architecture audit, CLI surfaces, integration harness | **DONE** |
| 24A | Web UI design specification | **DONE** |
| 24B | Legacy mock UI | **SUPERSEDED** by 24C |
| 24C | Clean-slate React/TS/Vite frontend | **ACCEPTED** by owner |
| 24D | Real API integration, identity, authorization, persistence and deployment hardening | **IMPLEMENTED / EXTERNAL CHECKS PENDING** |
| 25+ | Marketplace / JARVIS / further product | **DEFERRED** |

## Phase 24D completion

The current implementation record is [docs/PHASE24D.md](./PHASE24D.md), with deployment instructions in [docs/PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md).

Locally verified: full Go tests, vet and binary build; 17 frontend unit tests, TypeScript typecheck, lint and Vite production build. UI visuals were preserved.

External checks still pending: live PostgreSQL migrations/repository suite, actual container image build, production secrets/TLS setup, browser-level smoke and worker mTLS against a real CA. This machine has no Docker-compatible builder and no configured PostgreSQL service; those checks must not be reported as passing.

## Operational constraints

- Use one API replica until SSE pub/sub, login rate limiting and AI/autonomy safety limiting are shared across replicas.
- External process execution is disabled by default. Production must explicitly enable a binary allow-list; shell execution remains prohibited.
- A grant or assessment persisted as in-flight after a crash is an uncertain outcome. Reconcile before any retry; never automatically replay it.
- Go API is a long-running service, not a Vercel serverless function. Deploy the static React frontend independently and run Go API + workers on a persistent runtime.

## Historical roadmap notes

The original Phase 0–23 roadmap and architecture audit record what existed at those points in time. Historical statements about the mock executor being the production default, in-memory sessions/audit, or missing API integrations have been superseded by Phase 24D.
