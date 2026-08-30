# PHASE 24A — Web UI Architecture, UX & Design Specification

**MODE:** Design only · **HEAD baseline:** be855b9 · **No application code changes**

## 1. Executive summary

CYBEROS Web UI should be an **operator console**: thin, authenticated, project-scoped clients over existing `internal/api/v1` + services. The browser is **never** GrantIssuer, Broker, Trust authority, or Executor. All sensitive actions remain:

`Scope → Auth → CP → Decision → Grant → Broker → Host → Sandbox → Executor`

Remote: `Broker → RemoteHost → mTLS Worker → Assignment → Signed Result → Verify → Audit`

**Integrity ≠ Signature ≠ Trust ≠ Eligibility ≠ Authorization ≠ Execution** must be visible in UX labels and flows.

## 2. Existing architecture (CONFIRMED FROM CODE)

| Layer | Package / surface |
|-------|-------------------|
| API | `internal/api/v1` — health, version, capabilities, evaluate, grants, execute, projects, scopes, tasks, dashboard, assessments, findings, assets, risks, cases, reports, autonomy, AI, workers handlers, SSE |
| CLI | `internal/cli` — evaluate/grant/execute/status/doctor/trust-help/workers/assignments + domain commands |
| SDK | `internal/sdk` — `NewInMemory` / `NewProduction`; TrustAdmin on Production |
| Spine | controlplane, broker, execution (mtls, resultauth, assignment), tools (trust) |

## 3. UI security model

| Concept | UI rule |
|---------|---------|
| Auth | Session/API-key via server; no long-lived secrets in localStorage preferred; HttpOnly cookies or short-lived tokens |
| Authorization | Server-derived Decision/Grant only; never display a client “Authorized” toggle |
| Trust | Show server trust state; mutations only via SERVICE TrustAdmin API (not browser-as-admin by default) |
| Pack digest | Display-only from server; no edit of binding on Grant |
| Worker ID | Display from registry; never accept client override of mTLS identity |
| Result verified | Server `VerifyCompletion` only; no `verified=true` from client |

## 4. Information architecture (recommended routes)

| Route | Purpose | Sensitivity | Data source (CONFIRMED / GAP) |
|-------|---------|-------------|------------------------------|
| `/login` | Authenticate | High | Auth API (CONFIRM API key/session patterns) |
| `/dashboard` | Ops health | Med | `GET /api/v1/dashboard`, health, version |
| `/projects` | List projects | Med | `GET /api/v1/projects` |
| `/projects/:id` | Project overview | Med | projects + scoped lists |
| `/scopes` | Scopes | Med | `GET /api/v1/scopes` |
| `/tasks` | Tasks | Med | `GET /api/v1/tasks` |
| `/tasks/:id` | Task detail + evaluate path | High | tasks + evaluate/grant |
| `/evaluate` | CP evaluate form | High | `POST /api/v1/evaluate` |
| `/grants` | Grant list/detail | High | `POST /api/v1/grants` (list **API GAP**) |
| `/execute` | Submit Grant to Broker | Critical | `POST /api/v1/execute` |
| `/assessments` | Assessment lifecycle | Med | assessments routes |
| `/findings` `/assets` `/risks` `/cases` `/reports` | Domain | Med | existing handlers |
| `/workers` | Worker visibility | High | workers handlers (**PARTIAL**) |
| `/assignments` | Assignment lifecycle | High | **API GAP** (service exists; HTTP thin) |
| `/packs` `/trust` | Supply chain | Critical | tools/packs **PARTIAL**; TrustAdmin **API GAP** |
| `/audit` | Forensics | High | **API GAP** (audit service exists) |
| `/system` `/doctor` | Status | Low | health/version; doctor CLI parity **GAP** |

## 5. Dashboard widgets

| Widget | Type | Notes |
|--------|------|-------|
| Health / version | Operational | `/health`, `/version` |
| CP / Broker wired | Security | status-style; no secrets |
| Recent decisions / grants | Security | if list APIs exist |
| Workers ACTIVE count | Execution | no fingerprints secrets |
| Assignment PENDING/RUNNING | Execution | |
| Security events | Audit | observational |

## 6. Execution UX chain (visual)

```
Task → Evaluate → Decision → Grant → Execute(Broker) → [Local|Remote]
  Remote: Worker(mTLS) → Assignment → Signed Result → Verify → Audit
```

Never “Run shell”. Show PackContentDigest, GrantID, WorkerID, Result verification status from **server**.

## 7. Worker / Assignment / Pack / Audit UX

- **Worker:** ID, project, host, status badge (ACTIVE/DISABLED/REVOKED), fingerprint **public** only, last seen. No private keys.
- **Assignment:** ID, GrantID, PackContentDigest, worker, host, lease, status pipeline, result hash.
- **Pack/Trust:** digest, signature meta, trust state badges (UNTRUSTED/VERIFIED/TRUSTED/REVOKED). Mutations → TrustAdmin SERVICE only.
- **Audit:** timeline filters by project, correlation_id, grant_id, assignment_id — read-only.

## 8. API ↔ UI mapping (summary)

| UI action | API | Boundary |
|-----------|-----|----------|
| Evaluate | POST `/api/v1/evaluate` | CP |
| Issue grant | POST `/api/v1/grants` | Issuer after Decision |
| Execute | POST `/api/v1/execute` | Broker only |
| Catalog | GET projects/scopes/tasks | isolation middleware |
| Autonomy/AI | existing autonomy/ai routes | non-authoritative plans |

**API GAPs (do not implement in 24A):**

| Gap | Priority |
|-----|----------|
| TrustAdmin HTTP (trust/revoke pack, key lifecycle) | P1 |
| Assignment list/get/status HTTP | P1 |
| Audit event list/filter HTTP | P1 |
| Grant list by project | P2 |
| System doctor JSON endpoint | P2 |
| Pack inspect/digest HTTP completeness | P2 |

## 9. CLI ↔ UI parity

| Workflow | CLI | API | UI |
|----------|-----|-----|-----|
| evaluate/grant/execute | Yes | Yes | Spec |
| status/doctor | Yes | Partial | Spec |
| trust mutations | Help only | **GAP** | Deferred until API |
| workers/assignments | Help | Partial | Spec + gaps |

## 10. Design system

- **Tone:** security ops console (dense tables, monospace for IDs/digests, high-contrast status).
- **Statuses:** text + icon + badge (Healthy, Degraded, Denied, Revoked, Expired, Failed, Running, Pending, Unknown).
- **Confirmations:** required for revoke/execute; never one-click Grant mint without Decision display.
- **Desktop-first;** mobile: status/alerts only.

## 11. Frontend recommendation

**Vite + React + TypeScript + Tailwind (+ optional shadcn/ui)**

**Why:** static SPA or embed under Go (`internal/api/v1/static` already exists pattern); strong typing for Grant/Decision JSON; simple deploy beside Go binary; no server-side auth logic in Next needed if Go owns auth.

## 12. External frontend builder contract

- Routes/pages as §4 only; mock API client behind `api/` interface.
- Mock data: clearly tagged `MOCK_` never shipped as production trust.
- Components: `StatusBadge`, `DigestView`, `GrantCard`, `DecisionPanel`, `WorkerTable`, `AssignmentTimeline`, `AuditFeed`.
- No client-side “set trusted”; no local Grant forging.

## 13. Integration contract (future 24C)

```
Generated UI (static/) → fetch('/api/v1/...') → Go handlers → SDK/services → spine
```

Preserve: all Go security packages. Replaceable: static assets only. Env: API base URL, never embed private keys.

## 14. Threat model (UI)

| Threat | Defense |
|--------|---------|
| XSS | CSP, escape digests, no `dangerouslySetInnerHTML` for server strings |
| CSRF | SameSite cookies / CSRF token on mutating POSTs |
| IDOR | Server project isolation (existing isolation middleware) |
| Forged trust/Grant | Server rejects client EffectivePackTrust / Grant mint |
| Token theft | Short TTL, secure storage |
| Cache leakage | Cache-Control on sensitive GETs |

## 15. Testing strategy (future)

- Contract tests against API OpenAPI/snapshot  
- E2E: login → evaluate → grant → execute (mock Broker)  
- Negative: inject trusted=true → denied  

## 16. Manual verification plan (future)

1. Login with test principal  
2. Evaluate ALLOW → Grant → Execute mock  
3. Attempt pack B digest on Grant A path → DENY  
4. Confirm UI never shows private keys  

## 17. Deferred

- **24B** Visual frontend generation (external builder)  
- **24C** Integration into CYBEROS static/API  
- Marketplace / JARVIS — out of scope  

## FINAL SECURITY SPINE

```
Scope → Auth → CP → Decision → Grant → Broker → Host → Sandbox → Executor
Remote: Broker → RemoteHost → mTLS Worker → Assignment → Signed Result → Verify → Audit
```
