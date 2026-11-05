# CYBEROS Architecture Decisions

**Phase 0 Foundation**

## ADR-001: Primary Language

**Decision:** Go is the sole language of the CYBEROS core.

**Rationale:** Performance, concurrency model, static binary distribution, strong typing, and explicit control. Python (and other languages) may appear only at extension boundaries (plugins, AI/ML tooling, research scripts).

## ADR-002: Modular Monolith First

**Decision:** Start as a modular monolith with strong package boundaries. No microservices in Phase 0/1.

**Rationale:** Simpler operational model while the security invariant and domain model are still being proven. Boundaries are designed so extraction is possible later.

## ADR-003: Non-Bypassable Security Chain

**Decision:** Every sensitive operation must traverse the full chain ending in Tool Broker + Execution Grant. No shortcuts from CLI/API/Agent/ZOYA/UAB/Plugin directly to tools.

**Rationale:** Security is the primary invariant of the system. Convenience must never override it.

## ADR-004: PostgreSQL as Initial Store

**Decision:** PostgreSQL is the only database in early phases. Domain is isolated behind repository ports.

**Rationale:** Mature, transactional, good JSON support, widely available. Avoid multi-store complexity until needed.

## ADR-005: AI is Optional

**Decision:** CYBEROS must function completely without any AI provider. AI adapters are never core dependencies.

**Rationale:** Reliability, offline capability, and avoidance of vendor lock-in.

## ADR-006: Module Path

**Decision:** Go module path is `cyberos` for the foundation phase.

**Rationale:** Simple and self-contained. Can be changed to a full module path (e.g. github.com/…) later if the project is published.


## Phase 24D decisions
- **Grant provenance = HMAC-signed grants + durable issued-grant ledger.** `broker.GrantSigner` signs authoritative fields (proves origin cheaply, rejects forgery before any state is touched; key `CYBEROS_GRANT_SIGNING_KEY`). The ledger (`execution_grant_ledger`, migration 023) is the authority for one-time claim, revocation and expiry via a single atomic conditional UPDATE. Signature alone cannot express revocation or cluster-safe replay protection, and a ledger alone is weaker against a ledger rollback, so both are required. Issuance fails closed if the ledger cannot record. (Supersedes the earlier signature-only decision.)
- **Ownership comes from server-side records, never client headers.** A project/tenant named by a client is a claim to be verified. Foreign by-id reads return 404 (indistinguishable from missing); explicit cross-project declarations on writes return 403.
- **Roles are coarse permission levels** (viewer/analyst/admin/operator) layered under tenant/project ownership; tenant credentials are never operators. Persisting/issuing roles on API keys is an open item.
- **Sessions = server-side opaque tokens in an HttpOnly SameSite=Strict cookie + per-session CSRF header + Origin check**; login exchanges an API key. No credentials in localStorage.
- **Live is the default API mode; preview is explicit opt-in.** Unsupported surfaces throw `ApiError('unsupported')` rather than fake data.
- **Frontend is embedded** in the Go binary via `make web` into `internal/api/v1/static` (gitignored except PLACEHOLDER).

## Phase 24C frontend decisions
- **Design language "SPINE"**: dark-first, phosphor accent, semantic hue per spine link; hand-written CSS tokens, no Tailwind (instructions allowed CSS or Tailwind; fewer deps, no install needed).
- **Single API seam** `CyberosApi`; no grant/trust/verify/execute methods exist by design.
- **Preview adapter in one file** (`api/preview.ts`) to delete in 24D; UI labels preview mode everywhere.
- **Sign-in is inert** (disabled fields) rather than faking auth.
- **Derived views** (assessments, decisions, grants, execution, provenance, risk) built from existing fixtures to avoid inventing backend contracts; documented as deferred.
- **Static embed**: replaced old dashboard with a minimal placeholder `index.html` to satisfy `//go:embed static/*`.
- **Code-splitting**: every page lazy-loaded via `app/router.tsx`.
