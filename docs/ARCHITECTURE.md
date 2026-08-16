# CYBEROS Architecture

**Phase 0 Foundation Document**

## 1. High-Level System Architecture

```mermaid
flowchart TD
    HUMAN[Human Users]
    CLI[CLI]
    WEB[Web UI<br/>Future]
    ZOYA[ZOYA / JARVIS<br/>Future]
    UAB[UAB<br/>Future]
    AGENTS[Agents<br/>Future]

    HUMAN --> CLI
    HUMAN --> WEB
    ZOYA --> SDK
    UAB --> SDK
    AGENTS --> SDK

    CLI --> API
    WEB --> API
    SDK[SDK] --> API

    API --> APP[Application Layer]
    APP --> CP[Control Plane]

    CP --> AUTH[Authorization]
    AUTH --> POL[Policy]
    POL --> PERM[Permission]
    PERM --> RISK[Risk]
    RISK --> APPR[Approval]
    APPR --> GRANT[Execution Grant]

    GRANT --> BROKER[Tool Broker]

    BROKER --> TOOLS[Tools]
    BROKER --> PLUGINS[Plugins]
    BROKER --> AGENT_RT[Agent Runtime]

    TOOLS --> EVIDENCE[Evidence]
    PLUGINS --> EVIDENCE
    AGENT_RT --> EVIDENCE

    EVIDENCE --> PROV[Provenance]
    PROV --> EVENTS[Events]
    EVENTS --> AUDIT[Audit]
    AUDIT --> STORAGE[(Storage)]
```

## 2. Security Execution Chain (Non-Bypassable)

```mermaid
flowchart TD
    REQ[Request from CLI / API / SDK / Agent / ZOYA / UAB / Plugin]
    REQ --> SCOPE[Scope]
    SCOPE --> PRINCIPAL[Principal]
    PRINCIPAL --> AUTH[Authentication]
    AUTH --> AUTHORIZE[Authorization]
    AUTHORIZE --> POLICY[Policy]
    POLICY --> PERMISSION[Permission]
    PERMISSION --> RISK[Risk]
    RISK --> APPROVAL[Approval]
    APPROVAL --> GRANT[Execution Grant]
    GRANT --> BROKER[Tool Broker]
    BROKER --> EXEC[Execution]
    EXEC --> EVIDENCE[Evidence]
    EVIDENCE --> PROV[Provenance]
    PROV --> EVENT[Event]
    EVENT --> AUDIT[Audit]
```

**Invalid paths (forbidden):**

- CLI → Shell
- CLI → Tool
- API → Tool
- Agent → Tool
- ZOYA → Tool
- UAB → Tool
- Plugin → Tool
- Web UI → Tool

## 3. Module Dependency Direction

```mermaid
flowchart TD
    CONTRACTS[Contracts]
    DOMAIN[Domain]
    APPLICATION[Application]
    CONTROLPLANE[Control Plane]
    PORTS[Infrastructure Ports]
    ADAPTERS[Adapters]

    CONTRACTS --> DOMAIN
    DOMAIN --> APPLICATION
    APPLICATION --> CONTROLPLANE
    CONTROLPLANE --> PORTS
    PORTS --> ADAPTERS
```

Outer interfaces (CLI, API, SDK, Plugins, Agents, ZOYA, UAB) communicate through stable contracts only.

**Domain must never depend on:**

- PostgreSQL / SQL drivers
- HTTP frameworks
- CLI frameworks
- AI providers
- External security tools

## 4. Future Layers (Not Implemented in Phase 0/1)

- Plugin system (out-of-process, language-neutral)
- Agent runtime (constrained principals)
- ZOYA / JARVIS (orchestration, depends on CYBEROS)
- UniversalAIBridge (UAB)
- Provider-neutral AI adapters
- Distributed workers
- Multi-tenancy / cloud

## 5. Current Status

Phase 0 establishes only the skeleton, package boundaries, documentation, and build foundation. No business logic is present.
