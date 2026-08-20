# Extension Boundaries (Phase 3J)

| Concern | Package | Notes |
|---------|---------|--------|
| Capability discovery | `capability`, `plugin` | Discovery ≠ authorization |
| Tool adapters | `tools` | Register via Manager → execution.Registry |
| Language runtimes | `execution/runtime` | Labels only; process policy gates host |
| Process policy | `execution/process` | Fail-closed; allowlist; no shell default |
| Plugins (future load) | `plugin` | Prepare only; no Execute |
| Empty `adapters/*` | reserved | Future AI/FS/external — do not duplicate tools |

**Rule:** All execution enters through **Tool Broker → Sandbox → Executor**.
Python/Rust/TypeScript integrate as adapters or UI clients, not core rewrites.
