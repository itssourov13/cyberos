# CYBEROS Domain Model

**Phase 0 Foundation Document**

## Core Concepts (First-Class)

- **Project** — Top-level container for assets, scopes, and work.
- **Asset** — Target or resource under management (host, domain, application, etc.).
- **Scope** — Root authorization boundary. Constrains allowed assets, action classes, time windows, and status.
- **Task** — Unit of work. Always bound to a scope and principal.
- **Principal** — Who or what is acting (human, service, agent, plugin, ZOYA, etc.).
- **Capability** — What is technically available (e.g. `recon.domain.enumerate`).
- **Policy** — Rules that govern authorization decisions.
- **Permission** — Concrete allowance derived from policy + principal + scope.
- **Risk** — Risk evaluation associated with a requested action.
- **Approval** — Human or policy-driven approval state when required.
- **ExecutionGrant** — Short-lived, narrowly scoped authorization token for execution.
- **Evidence** — Raw or structured output produced by execution.
- **Finding** — Interpreted security result derived from evidence.
- **Provenance** — Full lineage: who, what, when, where, why, under which scope/task/capability, produced by which tool/agent.
- **Event** — Immutable record of something that happened.
- **Audit** — Security-relevant trail for accountability.
- **Report** — Aggregated view of findings and evidence (future).

## Conceptual Hierarchy

```
Project
 ├── Asset
 ├── Scope
 │    └── Task
 │         ├── Principal
 │         ├── Capability
 │         ├── Authorization
 │         ├── Policy
 │         ├── Permission
 │         ├── Risk
 │         ├── Approval
 │         └── ExecutionGrant
 │                └── Execution
 │                       ├── Evidence
 │                       ├── Finding
 │                       └── Provenance
 │
 └── Audit / Events
```

## Task States (Foundation)

Explicit states only. No arbitrary mutation.

- CREATED
- PENDING
- AUTHORIZED
- WAITING_APPROVAL
- READY
- RUNNING
- SUCCEEDED
- FAILED
- CANCELLED
- EXPIRED

Future: scheduling, retries, dependencies, DAG workflows, worker queues, distributed execution, agent-created tasks.

## Design Rules

- Do not reduce everything to generic JSON blobs.
- Domain entities must remain pure (no infrastructure dependencies).
- Repository interfaces live at the edge of the domain/application boundary.
- Adapters implement those interfaces.
