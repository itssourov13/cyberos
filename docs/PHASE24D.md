# Phase 24D — Backend integration (frontend ⇄ Go API)

Branch `feat/cyberos-phase-24d-integration`, based on the accepted 24C commit `e9cf797`. The SPINE UI (design system, routes, layouts, lazy loading) is unchanged except where real data required it.

## Frontend

- `frontend/src/api/http.ts` — single transport: same-origin `/api/v1`, `credentials: 'include'`, 15 s timeout + caller `AbortSignal`, `X-CSRF-Token` on non-GET, typed `ApiError` for 401/403/404/409/422/429/501/5xx/network/timeout. A 401 on a protected call fires the session-expired handler.
- `frontend/src/api/live.ts` — `CyberosApi` adapter over real routes only. No fixture fallback. Backend structs without JSON tags arrive PascalCase; mapping lives here. Tenant callers must name a project on most reads, so list calls fan out over the caller's projects.
- `frontend/src/api/session.ts`, `app/providers/SessionProvider.tsx`, `app/guards/RequireSession.tsx` — sign-in, restore-on-reload (`GET /auth/me`), sign-out, expiry, 401 handling, redirect-after-login limited to `/app…` paths. The route guard is UX only; the backend authorises every request.
- Sign-in takes a CYBEROS API key, exchanged once for an HttpOnly, SameSite=Strict session cookie. The key and cookie are never readable or stored by JS; only the in-memory CSRF token is held.
- Mode: **live is the default**. Preview fixtures load only with `VITE_CYBEROS_API_MODE=preview` (shown with a banner and "Preview data" chip). A dead backend shows an error, never fixtures.

## API endpoints used by the live adapter

`GET /health`, `/version`, `/dashboard`, `/projects`, `/scopes`, `/tasks`, `/capabilities`, `/findings`, `/assets`, `/cases`, `/reports`; `POST /grants`, `/execute`; `GET /auth/me`, `POST /auth/login`, `/auth/logout`.

## Backend

- Auth: `internal/auth/session.go` (server-side sessions keyed by SHA-256 of the token, TTL + idle expiry, CSRF token), `internal/api/v1/handlers_auth.go` (login rate-limited, origin check, `no-store`), middleware accepts the session cookie, requires same-origin + CSRF for cookie-authenticated writes.
- Principal binding: `bindPrincipal` takes identity from the authenticated context; a client-supplied `principal_id` that differs is 403.
- Isolation: tenant/project enforcement on projects, scopes, tasks, dashboard, capability registration (operator-only), worker routes, and SSE event filtering (see `tests/security/isolation_24d_test.go`).
- Grant provenance: `internal/broker/provenance.go` — `GrantSigner` HMAC-signs every authoritative grant field at issuance (`Signature`); the Broker rejects unsigned, tampered, foreign-key, revoked, expired and replayed grants; mutable `Status` is excluded and revocation is server-side. `CYBEROS_GRANT_SIGNING_KEY` (≥32 bytes, hex) is required in production; development uses an ephemeral key. The execute handler also requires the caller to own the grant.
- Serving: `internal/api/v1/ui.go` embeds `static/` (`make web` copies `frontend/dist`). `/api/*` never returns `index.html`; missing assets → 404; traversal blocked; hashed assets immutable-cached; CSP hashes the inline bootstrap script; security headers set. Unbuilt binaries answer 503 with build instructions.

## Not done (exact dependencies)

| Area | Gap | Needed |
|---|---|---|
| Workers list | `GET /projects/{p}/workers` is a stub; `WorkerRegistry` has no `List` | add `List` to registry + durable store, wire handler; then add `workers()` mapping (`mapWorker` is ready in `live.ts`) and suspend/revoke buttons |
| Assignments, Packs/trust, Audit feed, Decisions/Grants/Execution/Evidence/Provenance listings | no read endpoints | read endpoints over existing repositories (decisions, grants, evidence, trust store, audit log) |
| Dashboard aggregates | grants/denials 24h, throughput, fleet counts not exposed | aggregate endpoints; UI shows "—" until then |
| Asset exposure / risk score | not on asset model | risk endpoints return per-asset evaluations; add a summary route |
| Report download, assessment creation, case status, autonomy, events UI | backend routes exist; UI controls not wired | follow-up wiring |
| Durable sessions | sessions are in-memory (re-login after restart) | persist session store |
| Frontend unit tests | no test runner in `frontend/` | add Vitest for `http.ts` / session mapping |
| Browser E2E | not run | — |
