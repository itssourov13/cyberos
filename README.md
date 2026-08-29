# CYBEROS

**Cybersecurity Operating System**

CYBEROS is a long-term cybersecurity platform and operating environment designed for security researchers, penetration testers, red/blue teams, SOC analysts, vulnerability researchers, bug bounty hunters, security engineers, automation engineers, students, human operators, and autonomous security agents.

This is **not** merely a scanner, pentesting toolkit, dashboard, or AI chatbot. It is intended to become a complete, secure, auditable control plane for cybersecurity work.

## Status

**Phase 23C — Documentation & live verification harness** (current HEAD)

Security spine through Phase 22 is implemented and unit-tested. CLI operator surfaces (23B) available.
Live PostgreSQL / mTLS worker verification: **SKIPPED unless `DATABASE_URL` / cert env is set**.

### Security spine (authoritative)

```
Scope → Auth → Control Plane → Decision → Grant → Broker → Host → Sandbox → Executor
```

Remote: `Broker → RemoteHost → mTLS Worker → Assignment → Signed Result → Verify → Audit`

- Broker is the **sole execution gateway**
- TRUSTED ≠ AUTHORIZED; PackKeyID is provenance only
- CLI/API are operator surfaces, not authorization kernels
- AI/Event cannot mint Grants or call Broker

## Core Principles

- **Go-first core** — The CYBEROS core is written in Go. Other languages may appear only at extension boundaries (plugins, AI adapters, research tools).
- **Non-bypassable security chain** — Every sensitive operation must pass through:

  ```
  Scope → Principal → Authentication → Authorization → Policy →
  Permission → Risk → Approval → Execution Grant → Tool Broker →
  Execution → Evidence → Provenance → Event → Audit
  ```

- **Modular monolith** — Strong module boundaries now; extractable later.
- **API-first / CLI-first** — Thin clients only. Business logic lives in the core.
- **AI-optional** — The platform functions completely without AI. AI providers are never core dependencies.

## Quick Start (Foundation)

```bash
# Build
make build

# Version
./bin/cyberos version

# Run placeholder
./bin/cyberos
```

## Project Layout

```
cyberos/
├── cmd/cyberos/          # Main binary entrypoint
├── internal/             # Private implementation (not for external import)
│   ├── contracts/        # Stable internal contracts
│   ├── domain/           # Core domain entities
│   ├── application/      # Application services
│   ├── controlplane/     # Authorization, policy, risk, approval
│   ├── identity/         # Principals
│   ├── taskengine/       # Task state machine (foundation)
│   ├── capability/       # Capability registry
│   ├── broker/           # Tool Broker
│   ├── execution/        # Executors (mock only in early phases)
│   ├── evidence/         # Evidence handling
│   ├── findings/         # Findings
│   ├── provenance/       # Provenance
│   ├── events/           # Event model
│   ├── audit/            # Audit
│   ├── storage/          # Repository ports + adapters
│   ├── api/              # HTTP API handlers
│   ├── cli/              # CLI command structure
│   ├── sdk/              # SDK foundation
│   ├── config/           # Configuration
│   └── adapters/         # External adapters
├── pkg/version/          # Public version info
├── api/                  # OpenAPI / schemas
├── docs/                 # Architecture & design documentation
├── state/                # Project state tracking
├── tests/                # Unit, integration, security, API, contract tests
├── migrations/           # Database migrations (future)
├── configs/              # Static configuration files
└── scripts/              # Helper scripts
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | High-level system architecture |
| [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md) | Security invariant and control plane |
| [docs/DOMAIN_MODEL.md](docs/DOMAIN_MODEL.md) | Core domain concepts |
| [docs/API.md](docs/API.md) | API boundary |
| [docs/EVENTS.md](docs/EVENTS.md) | Event model |
| [docs/PLUGINS.md](docs/PLUGINS.md) | Future plugin architecture |
| [docs/AGENTS.md](docs/AGENTS.md) | Future agent runtime |
| [docs/ZOYA.md](docs/ZOYA.md) | Future ZOYA / JARVIS layer |
| [docs/UAB.md](docs/UAB.md) | UniversalAIBridge integration |
| [docs/STORAGE.md](docs/STORAGE.md) | Storage architecture |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Deployment notes |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Development guide |

## Build Plan

See [state/PHASE_1.md](state/PHASE_1.md) and the master build plan. Construction is strictly dependency-driven:

Foundation → Contracts → Domain → Persistence → Control Plane → … → Full Platform

## License

See [LICENSE](LICENSE).
