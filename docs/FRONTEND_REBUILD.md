# Frontend Rebuild — Phase 24C

Clean-slate replacement of the demo UI. **Status: implementation complete for the Phase 24C scope; audit findings have been fixed; owner re-verification pending** (see "Verification" below).

## What was retired

| Location | Action |
|---|---|
| `frontend/src/` (old demo) | Removed: old `App`, pages, layouts, `index.css`, mock client, demo components |
| `web/ui/` | Removed (`app.js`, `index.html`, `styles.css`) |
| `internal/api/v1/static/` | Old dashboard removed. Replaced by a minimal placeholder `index.html` so `//go:embed static/*` in `internal/api/v1/ui.go` still compiles |

No Go handlers, security packages, contracts, migrations or tests were touched.

## New stack

TypeScript · React 19 · Vite 8.3.5 (Node `^20.19 || >=22.12`) with `@vitejs/plugin-react` 6.2.1 · React Router 7 · lucide-react · hand-written CSS design tokens (no Tailwind, no extra runtime deps).

## Brand and design language: "SPINE"

- Identity built around the execution spine (Scope → Decision → Grant → Broker → Execution → Evidence), rendered as the signature `SpineFlow` component and the logo mark.
- Dark-first with a light theme; one phosphor accent plus semantic hues per link of the chain (grant = amber, trust = indigo, evidence = cyan).
- Motion: staged reveals, hero spine walkthrough, pulsing active stage. All motion is neutralised by `prefers-reduced-motion`.
- Accessibility: skip links, focus-visible rings, ARIA on tabs/segmented/meter/dialog, sr-only outcome text on spine nodes, keyboard-operable tables.

## Structure (`frontend/src`)

```
main.tsx                    entry, imports design CSS
design/                     tokens.css · base.css · ui.css · layout.css
api/                        types.ts (view models) · client.ts (CyberosApi seam) · preview.ts (fixtures) · index.ts (ApiProvider/useApi)
app/                        router.tsx · nav.ts · providers/ · layouts/ (SiteLayout, AppShell, ThemeToggle)
components/ui/              primitives, DataTable, Drawer, Async, Charts
components/domain/          badges, SpineFlow, AuditSlice
components/site/            LiveSpine (marketing illustration)
components/brand/           Logo
lib/                        format, useAsync, spine
pages/site/                 HomePage, MarketingPages (Platform, Architecture, Security)
pages/auth/                 SignInPage
pages/app/                  Dashboard, Engagement, Intel, Control, Proof, Fleet, Platform pages
```

## Routes

Public: `/`, `/platform`, `/architecture`, `/security`, `/sign-in` (`/login` redirects).
Console (`/app/...`): dashboard, projects, projects/:id, scopes, tasks, assessments, assets, findings, risk, cases, reports, evaluate, decisions, grants, execution, evidence, provenance, workers, assignments, packs, autonomy, events, system, settings.

## Security boundaries in the UI

- `CyberosApi` has **no** methods to issue grants, change trust, verify results or run tools. The only write-ish call is `submitSpineRequest`, which *asks* the backend.
- Grants page has no "issue" control; Packs page has no trust toggle; Workers drawer has no suspend/revoke (documented as backend-pending).
- Sign-in is **not** implemented: fields are disabled and no session is created. "Continue to preview console" just navigates.
- Preview mode is signalled by a topbar chip and a persistent banner; simulated traces are labelled.

> **Phase 24D update:** a live adapter (`api/live.ts`) is now the default and preview data is opt-in via `VITE_CYBEROS_API_MODE=preview`. See `docs/PHASE24D.md`. Phase 24C is accepted.

## Preview-only derivations (replace in Phase 24D)

| View | Currently derived from |
|---|---|
| Assessments | findings grouped by `assessmentId` |
| Decisions / Grants / Execution / Provenance / Events | `audit()` stream filtered by action prefix |
| Risk ranking | local severity × exposure weighting (display only) |

## Preview adapter behaviour (post-audit)

- **Scope validation:** `simulate()` now checks the target against the scope's declared targets: exact domain/host, strict `*.` wildcard boundary (apex not covered), IPv4/IPv6 CIDR containment, exact repo and cloud-account identifiers. Anything unlisted is denied at the Scope stage. Results remain labelled simulated.
- **Identifiers:** preview digests are `sample:<hex>` placeholders (not `sha256:`). The `Digest` component renders them with a dashed style and tooltip, and the Evidence page shows a warning banner in preview mode.

## Verification

The owner performs typecheck, lint, build and tests separately. A previous audit reported TypeScript errors (generic inference in `Tabs`/`Segmented`/`DataTable` call sites) and Oxlint warnings (JSX keys, effects that set state, mixed exports); these were addressed by explicit generics, `key` props, deriving nav-open state from the pathname, and moving context/hook/constant exports into `theme-context.ts`, `toast-context.ts` and `spine-stages.ts`.

`Tests/build/lint/type-check were NOT executed because the owner reserved verification for a separate phase.`
