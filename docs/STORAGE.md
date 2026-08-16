# CYBEROS Storage Architecture

**Phase 1C — Persistence (Implemented)**

## Layering

```
Domain
  ↓
Repository Interfaces (internal/storage/ports)
  ↓
PostgreSQL Adapter (internal/storage/postgres)
```

Domain code never imports SQL drivers or PostgreSQL types.

## Initial Schema (migration 001_init.sql)

Tables:
- `projects`
- `principals`
- `assets` (FK → projects)
- `scopes` (FK → projects)
- `capabilities`
- `tasks` (FK → projects, scopes, principals, capabilities)
- `schema_migrations`

## Configuration

```
CYBEROS_DATABASE_URL=postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable
```

## Migrations

SQL files under `migrations/` applied in lexical order via `postgres.ApplyMigrations`.

## Tests

Integration tests (build tag `integration`) exercise create/get/update round-trips against a real PostgreSQL instance.
