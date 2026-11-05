# CYBEROS Build Plan

**Phase 0 Foundation Document**

## High-Level Sequence

```
FOUNDATION
    ↓
CONTRACTS
    ↓
DOMAIN
    ↓
PERSISTENCE
    ↓
CONTROL PLANE
    ↓
TASK ENGINE
    ↓
CAPABILITY REGISTRY
    ↓
EXECUTION GRANTS
    ↓
TOOL BROKER
    ↓
EVIDENCE / FINDINGS
    ↓
PROVENANCE
    ↓
EVENTS / AUDIT
    ↓
API
    ↓
CLI
    ↓
SDK
    ↓
PLUGIN SYSTEM
    ↓
AGENT RUNTIME
    ↓
WEB UI
    ↓
ZOYA / JARVIS
    ↓
UAB
    ↓
ADVANCED AI
    ↓
DISTRIBUTED EXECUTION
    ↓
MULTI-TENANCY / CLOUD
    ↓
FULL CYBERSECURITY PLATFORM
```

## Phase Roadmap Diagram

```mermaid
flowchart TD
    P0[Phase 0<br/>Foundation]
    P1[Phase 1<br/>Secure Core]
    P2[Phase 2<br/>Task + Execution]
    P3[Phase 3<br/>Plugins + Tools]
    P4[Phase 4<br/>Web Platform]
    P5[Phase 5<br/>Agent Runtime]
    P6[Phase 6<br/>ZOYA / JARVIS]
    P7[Phase 7<br/>UAB]
    P8[Phase 8<br/>AI Ecosystem]
    P9[Phase 9<br/>Distributed Execution]
    P10[Phase 10<br/>Cloud + Multi-Tenancy]

    P0 --> P1
    P1 --> P2
    P2 --> P3
    P3 --> P4
    P4 --> P5
    P5 --> P6
    P6 --> P7
    P7 --> P8
    P8 --> P9
    P9 --> P10
```

## Dependency Graph (Core)

```mermaid
flowchart TD
    CONTRACTS[Contracts]
    DOMAIN[Domain]
    REPOS[Repositories]
    CP[Control Plane]
    PG[PostgreSQL Adapter]
    TE[Task Engine]
    CAP[Capability Registry]
    GRANT[Execution Grant]
    BROKER[Tool Broker]
    EVIDENCE[Evidence]
    FINDING[Finding]
    AUDIT[Audit]
    EVENTS[Events]
    API[API]
    CLI[CLI]
    SDK[SDK]

    CONTRACTS --> DOMAIN
    DOMAIN --> REPOS
    DOMAIN --> CP
    REPOS --> PG
    CP --> TE
    TE --> CAP
    CAP --> GRANT
    GRANT --> BROKER
    BROKER --> EVIDENCE
    BROKER --> FINDING
    BROKER --> AUDIT
    EVIDENCE --> EVENTS
    FINDING --> EVENTS
    AUDIT --> EVENTS
    EVENTS --> API
    EVENTS --> CLI
    EVENTS --> SDK
```

## Rules

- Do not implement modules out of order.
- Do not introduce real offensive tools or unrestricted execution before the control plane and Tool Broker are solid.
- Stop and validate at the end of each major phase.

## Phase 24C — Frontend rebuild milestones
1. [x] Retire old UI in `frontend/src`, `web/ui`, `internal/api/v1/static` (placeholder kept for embed)
2. [x] Design tokens, base/ui/layout CSS, primitives, DataTable, Drawer, Async, charts, badges, SpineFlow, logo
3. [x] API seam + preview adapter; providers (theme, toast, api)
4. [x] Router, SiteLayout, AppShell, nav model
5. [x] Public site (home, platform, architecture, security) and sign-in UI
6. [x] Console pages: dashboard, engagements, intelligence, control plane, proof, fleet/supply, platform
7. [x] Documentation + handoff
8. [x] Post-audit fixes (principal binding, listen safety, Vite 8.3.5, TS inference, preview scope validation, sample identifiers, lint)
8b. [ ] Owner re-verification (typecheck, lint, build, go tests) and acceptance
9. [~] Phase 24D integration — partial; see docs/PHASE24D.md (auth, grant provenance, isolation, live adapter, embedded serving done; workers/packs/audit/decision-listing endpoints pending)
