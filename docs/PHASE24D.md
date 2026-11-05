# Phase 24D — Backend integration (frontend ⇄ Go API)

Branch `feat/cyberos-phase-24d-integration`, based on the accepted 24C commit `e9cf797`. The SPINE UI (design system, routes, layouts, lazy loading) is unchanged except where real data required it.

Status: **PARTIAL.** Each item below is labelled *verified* (a test or run in this repo exercised it) or *unverified*. Nothing here has been exercised in a browser or against a live PostgreSQL server.

## Frontend

- `frontend/src/api/http.ts` — single transport: same-origin `/api/v1`, `credentials: 'include'`, 15 s timeout + caller `AbortSignal`, `X-CSRF-Token` on non-GET, typed `ApiError` for 401/403/404/409/422/429/501/5xx/network/timeout. A 401 on a protected call fires the session-expired handler.
- `frontend/src/api/live.ts` — `CyberosApi` adapter over real routes only. No fixture fallback. Backend structs without JSON tags arrive PascalCase; mapping lives here. Tenant callers must name a project on most reads, so list calls fan out over the caller's projects.
- `frontend/src/api/session.ts`, `app/providers/SessionProvider.tsx`, `app/guards/RequireSession.tsx` — sign-in, restore-on-reload (`GET /auth/me`), sign-out, expiry, 401 handling, redirect-after-login limited to `/app…` paths. The route guard is UX only; the backend authorises every request.
- Sign-in takes a CYBEROS API key, exchanged once for an HttpOnly, SameSite=Strict session cookie. The key and cookie are never readable or stored by JS; only the in-memory CSRF token is held.
- Mode: **live is the default**. Preview fixtures load only with `VITE_CYBEROS_API_MODE=preview` (banner + "Preview data" chip). A dead backend shows an error, never fixtures.
- Verified: `npm run typecheck`, `npm run lint` (0 warnings), `npm run build`. Unverified: rendering of every screen against a live backend.

## API endpoints used by the live adapter

`GET /health`, `/version`, `/dashboard`, `/projects`, `/scopes`, `/tasks`, `/capabilities`, `/findings`, `/assets`, `/cases`, `/reports`; `POST /grants`, `/execute`; `GET /auth/me`, `POST /auth/login`, `/auth/logout`.

## Authentication and sessions (verified by `tests/security/session_test.go`)

- `internal/auth/session.go`: server-side sessions keyed by SHA-256 of an opaque token, absolute + idle expiry, per-session CSRF token.
- `internal/api/v1/handlers_auth.go`: login is rate-limited, origin-checked, `no-store`. The middleware accepts the session cookie and requires same-origin + CSRF for cookie-authenticated writes.
- Principal binding: `bindPrincipal` takes identity from the authenticated context; a client-supplied `principal_id` that differs is 403.
- Sessions are in memory (restart = sign in again).

## Authorization and isolation (24D-1; verified by `tests/security/isolation_24d_test.go`, `internal/api/v1` tests)

- Ownership is resolved from **server-side records**, never from client headers: the caller's tenant comes from the authenticated identity, a project's tenant from the catalog, and an assessment/case/finding/asset/risk's project from the stored resource. A project id in a query/header/body is only a claim that is checked (`authorizeProject` in `internal/api/v1/isolation.go`).
- Tenant callers fail closed on unknown/unassigned projects. Foreign **by-id reads return 404**, identical to a missing id, so ids cannot be probed (`enforceObject`). Explicit cross-project declarations on writes remain 403.
- Scope and task references on evaluate/grants/assessments/workflows/autonomy are validated to belong to the named project (400 otherwise).
- Roles (`internal/auth/roles.go`): `viewer`, `analyst`, `admin`, `operator`; permission checks via `requirePermission` on evaluate (write), grants/execute (execute), case transitions/acceptance, worker admin, autonomy control/approve, trust admin, audit read. A tenant-less credential is the platform operator; tenant credentials are never operators. Capability registration and trust administration are operator-only.
- Domain errors map to statuses in one place (`internal/api/v1/errors.go`); 5xx messages are generic so driver details never reach clients.
- Cases: attributable, durable status history (`case_status_history`, migration 022, `GET /cases/{id}/history`); `Insert` never overwrites another tenant's case; `Transition` is atomic and conflict-aware.
- SSE streams filter events by the caller's tenant/project.
- **Known gap:** API-key roles are not persisted or issuable. `APIKeyAuthenticator.Issue` takes no role and the PostgreSQL key store has no role column, so every tenant key behaves as `analyst`; `viewer`/`admin` exist in code only (the in-memory store can carry a role in tests). Admin-gated actions (worker admin, autonomy approval, case acceptance) are therefore reachable only by the operator credential until a role column + issuance path is added.

## Grant provenance (verified by `internal/broker/provenance_test.go`, `ledger_test.go`)

Two layers, both required:

1. **Signature** (`internal/broker/provenance.go`): `GrantSigner` HMAC-signs every authoritative grant field at issuance. The Broker rejects unsigned, tampered and foreign-key grants before touching any state. Mutable `Status` is excluded. `CYBEROS_GRANT_SIGNING_KEY` (≥32 bytes, hex) is required in production; development uses an ephemeral key.
2. **Ledger** (`internal/broker/ledger.go`, PostgreSQL `internal/storage/postgres/grant_ledger.go`, migration 023): the Issuer records every grant (issuance fails closed if recording fails). The Broker claims it with one atomic conditional UPDATE, so unissued, replayed, revoked and expired grants are rejected across processes; revocation is visible to every instance. Execution outcome is recorded.

The execute handler also requires the caller to own the grant. Unit tests cover: legitimate issue→execute→replay-rejected, validly signed but unregistered grant, cross-instance revocation, expiry, 16-way concurrent claim (exactly one winner), issuance failing when the ledger is down. The PostgreSQL ledger itself is covered by an integration test (`-tags integration`) that **has not been run** (no PostgreSQL server was available); it compiles and passes `go vet`.

## Production serving (verified by `internal/api/v1/ui_test.go` and a manual smoke of the embedded binary)

`internal/api/v1/ui.go` embeds `static/` (`make web` copies `frontend/dist`). `/api/*` never returns `index.html`; missing assets → 404; traversal blocked; hashed assets immutable-cached; CSP hashes the inline bootstrap script; security headers set. Unbuilt binaries answer 503 with build instructions.

## Remaining work (exact dependencies)

| Area | Gap | Needed |
|---|---|---|
| API-key roles | not persisted/issuable (see above) | role column + migration, `Issue(..., role)`, CLI/admin path |
| Workers list | `GET /projects/{p}/workers` is a stub; `WorkerRegistry` has no `List` | add `List` to registry + durable store, wire handler; then `workers()` mapping (`mapWorker` ready in `live.ts`) and suspend/revoke buttons |
| Assignments, Packs/trust, Audit feed, Decisions/Grants/Execution/Evidence/Provenance listings | no read endpoints | read endpoints over existing repositories, authorised per project |
| Dashboard aggregates | grants/denials 24h, throughput, fleet counts not exposed | aggregate endpoints; UI shows "—" until then |
| Asset exposure / risk score | not on asset model | per-asset evaluation summary route |
| Report download, assessment creation, case status, autonomy, events UI | backend routes exist; UI controls not wired | follow-up wiring |
| Durable sessions | in-memory | persist session store |
| Frontend unit tests | no test runner in `frontend/` | Vitest for `http.ts` / session / mapping |
| PostgreSQL verification | ledger, case history, migrations 022/023 never run against a live DB | run `go test -tags integration ./internal/storage/postgres/` with `CYBEROS_DATABASE_URL` |
| Browser E2E | not run | — |
