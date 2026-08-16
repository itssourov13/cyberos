# CYBEROS Development Guide

**Phase 0 Foundation Document**

## Prerequisites

- Go 1.22+ (currently using 1.22.2)
- Git
- Make (optional but recommended)

PostgreSQL is required from Phase 1 onward for persistence.

## Building

```bash
make build          # produces bin/cyberos
make version        # print version info
make test           # run tests
make fmt
make vet
make mod-tidy
```

## Package Rules

- `cmd/` — executable entry points only
- `internal/` — all private implementation
- `pkg/` — only genuinely reusable public libraries (currently just `version`)
- Public integration surface is API + SDK + Contracts + Events, **not** direct Go package imports of internal code

## Dependency Direction

Contracts → Domain → Application → Control Plane → Ports → Adapters

Never invert this direction.

## Testing

```
tests/
├── unit/
├── integration/
├── security/     # critical invariant tests
├── api/
└── contracts/
```

Security tests protecting the non-bypassable chain are mandatory as soon as the control plane exists.

## State Tracking

Project progress, decisions, and handoff notes live under `state/`.
