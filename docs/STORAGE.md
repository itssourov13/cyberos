# CYBEROS Storage Architecture

**Phase 0 Foundation Document**

## Initial Choice

PostgreSQL.

## Layering

```
Domain
  ↓
Repository Interfaces (ports)
  ↓
PostgreSQL Adapter
```

Domain code must never import SQL drivers or PostgreSQL-specific types.

## Future Possibilities

- SQLite for local / single-user mode
- S3-compatible object storage for large evidence
- Specialized stores as needed

Do **not** introduce multiple databases in Phase 0/1.

## Migrations

SQL migrations will live under `migrations/`.  
Managed via a migration tool (to be selected in Phase 1).
