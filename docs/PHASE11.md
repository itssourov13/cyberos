# Phase 11 — Extensible Tool/Pack Ecosystem (**FROZEN** at `v0.11.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **11A** | Extension contract versioning (`ContractVersion` / V1) | Done |
| **11B** | Declarative parameter / argv / output schemas | Done |
| **11C** | PackFile loader, install, discovery catalog | Done |
| **11D** | Runtime, deps, config/secret refs, resource/health metadata | Done |
| **11E** | Final audit + freeze | Done |

## Extensible pack model

```
PackFile (JSON)
  → Loader (validate atomic, discovery only)
  → PackRegistry + Catalog
  → CapabilityResolver (discovery)
  → Workflow capability steps
  → ★ CP → Grant → Broker → Sandbox → Executor ★
  → Evidence / Provenance
```

**Pack installation ≠ authorization.**  
**Capability discovery ≠ grant.**  
**Schemas never execute.**

## Security invariants

- Sole execution path: Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor
- Process fail-closed; shell disabled by default
- No shell argv construction; ValidateArgv authoritative
- Secret refs only (no values in manifests/catalog)
- ResourceHints cannot weaken Sandbox
- Dependencies never auto-download

## Scalability (honest)

| Scale | Status |
|-------|--------|
| 10–100 tools | Supported (in-memory registry + files) |
| 1,000+ | Operational: durable catalog + pack distribution (future product work) |
| 10,000+ | Same; not an architectural spine blocker |

## Future compatibility

- **Kali/BlackArch:** map local binaries via pack binary metadata + operator allowlist
- **Python/Rust/native:** same process boundary
- **Remote workers:** future executor backends under Broker only
- **JARVIS:** read catalog/schemas/workflows; never issue grants or call Broker with arbitrary authority

## PostgreSQL

No Phase 11 pack catalog table (in-memory + file load). Live PG for other domains: **SKIP if DSN absent ≠ verified**.

## Known limitations

1. Pack catalog not durable across process restart without reloading files.
2. Operators must configure process allowlists separately.
3. No marketplace, signing, or automatic dependency install.
