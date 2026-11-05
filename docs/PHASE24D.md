# Phase 24D — Production integration and hardening

**Branch:** \`feat/cyberos-phase-24d-integration\`
**Code status:** Implemented; local Go and frontend verification passed.
**Deployment status:** Not deployed or externally verified against a live PostgreSQL service.

The existing SPINE visual system, layouts, routes, typography, spacing and component styling have been preserved. Phase 24D changes the application/API/data path, not the visual design.

## Delivered

### Frontend and session boundary

- \`frontend/src/api/http.ts\` is the single live transport: credentialed requests, HttpOnly session cookie, in-memory CSRF token, timeout/abort support, typed API errors, no fixture fallback.
- Live API mode is the default. Preview fixtures remain an explicit opt-in.
- Sign-in, session restore, logout, expiry handling and route guards use the backend session endpoints.
- Cross-origin browser calls are restricted to explicitly allow-listed origins. CORS uses exact scheme/host matching and credentialed responses; wildcard origins are never emitted.
- React Router SPA fallback is configured for Vercel. The Go binary continues to serve the embedded Vite build.

### Authentication, authorization and tenant isolation

- Production-like environments (anything except explicit development) require PostgreSQL, authentication, secure cookies, a persistent PostgreSQL session backend, a production signing secret and verified PostgreSQL TLS (\`sslmode=verify-full\`).
- Browser sessions store only a SHA-256 digest of the opaque token. Every lookup checks the active API-key row, expiry and current role. API-key revocation invalidates its sessions.
- API-key roles are persisted: \`viewer\`, \`analyst\`, \`admin\`, \`operator\`. Tenant-scoped credentials cannot be platform operators. Unknown or inconsistent role bindings fail closed.
- Bootstrap is a tenant-less \`bootstrap\` platform operator. Supply \`CYBEROS_BOOTSTRAP_API_KEY\` through a secret manager for first initialization; production never prints or silently generates a bootstrap secret.
- Assessment create/plan/run permissions are separated (write vs execute). Resource ownership is resolved from server-side records, not trusted client claims.
- Request body size limits, login/API safety rate limits, readiness checks and graceful HTTP shutdown are wired.

### Grants, execution and evidence

- Execution grants are HMAC-signed and backed by the durable PostgreSQL grant ledger. Issuance, one-time claim, completion and revocation are persisted.
- Broker completion errors propagate. Expired terminal grants may be cleaned up; uncertain \`CLAIMED\` records are retained for reconciliation and are never automatically replayed.
- \`POST /api/v1/grants/{id}/revoke\` exposes durable revocation. It requires an operator, or the tenant grant owner/project admin. A CLAIMED grant cannot be revoked.
- \`POST /api/v1/execute\` now runs through the execution lifecycle controller. It persists an execution record, evidence and audit/provenance links. Repeating the same grant returns the evidence-backed prior result instead of executing the target again.
- Production registers real tool adapters, not mock/stub executors. There is no implicit default executor. Production grant issuance requires an explicitly registered executor hint.
- Nmap and HTTP probing are target-bound. Their grants bind a canonical project asset and exact target string; Broker rejects a changed target before execution.
- Process execution is disabled by default. Enable only with an explicit binary allow-list; shell execution remains prohibited. Stdout/stderr are bounded while the child process is running.

### Durable data and runtime composition

- PostgreSQL-backed sessions, audit feed, project/tenant ownership, assets, findings, evidence, provenance, risk evaluations, cases, reports, workers, assignments, assessments, workflow runs, execution records and grant lifecycle are wired into the production composition.
- Project catalog refresh is paginated and atomically replaces the snapshot only after all repository reads succeed.
- Assessment state and per-step results persist. A PostgreSQL compare-and-set claim prevents multiple replicas from starting the same PLANNED assessment. A persisted RUNNING assessment after a crash is marked uncertain and blocked from automatic replay.
- Trust administration is exposed through operator-only routes for project pack trust/revocation and signer registration/inspection/disable/revoke. Mutations are audited.
- Worker heartbeats bypass human API-key middleware only on the heartbeat route, require a verified client certificate, and bind the SHA-256 certificate fingerprint to an active registered worker. The controller can use TLS 1.3 with a dedicated worker CA.
- \`/api/v1/health\` is liveness; \`/api/v1/ready\` checks PostgreSQL availability.

### Operational/deployment packaging

- \`Dockerfile\`: multi-stage Vite + Go build, Go 1.27.3 build image, Alpine 3.24.4 runtime, current CA bundle, non-root UID 10001, embedded frontend and migration SQL files.
- \`.dockerignore\` prevents secrets, local audit files, dependencies and build output entering the context.
- \`frontend/vercel.json\` provides a React Router SPA fallback.
- \`docs/PRODUCTION_DEPLOYMENT.md\` documents Vercel frontend + long-running Go API + managed PostgreSQL topology, secrets, bootstrap, grant revocation, pack trust, worker mTLS and tool allow-list operations.

## Verification completed in the working tree

| Check | Result |
|---|---|
| \`gofmt\` on all changed/new Go files | Pass |
| \`go test ./... -count=1\` | Pass |
| \`go vet ./...\` | Pass |
| \`go build ./cmd/cyberos\` | Pass |
| \`git diff --check\` | Pass |
| Frontend Vitest | 17/17 pass |
| Frontend TypeScript typecheck | Pass |
| Frontend oxlint | 0 warnings, 0 errors |
| Frontend Vite production build | Pass |
| API lifecycle execution regression | Pass: evidence persists and same grant retry does not re-execute |

## External verification still required before public traffic

These are environment checks, not unresolved source-code tasks:

1. Apply migrations 001–025 against a disposable PostgreSQL instance using the same \`sslmode=verify-full\` and CA configuration intended for production. Run the PostgreSQL integration suite and verify migration rollback/restore.
2. Build the Docker image in a host with Docker/BuildKit. This development environment has no Docker, Podman, Buildah or Nerdctl binary, so the image itself was not built here.
3. Configure production secrets, TLS certificates, worker CA, database CA and real allowed tool paths. Process execution is intentionally disabled until explicitly enabled.
4. Perform operator-owned live smoke checks: bootstrap, sign-in/logout, key issue/revoke, cross-tenant denial, trust mutation/audit, grant issue/revoke/replay, one controlled executor run, worker certificate rejection/acceptance, SSE and readiness failure.
5. Keep one API replica until SSE pub/sub, login limiting and AI/autonomy safety limiting use shared distributed services. Those mechanisms are currently process-local.
6. No browser E2E run was performed. The frontend typecheck/lint/build and API tests passed, but pixel/render behaviour against a live API has not been externally exercised.

**Conclusion:** Phase 24D implementation is complete in this branch and the available local code/test/build checks pass. It is not represented as deployed or infrastructure-verified production. Do not call the service live until the external checks above are completed.
