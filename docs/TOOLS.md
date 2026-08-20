# Security Tool Adapters

## First real adapter: Nmap (Phase 6A)

| Field | Value |
|-------|--------|
| Package | `internal/tools/nmap` |
| Tool ID | `tool.nmap` |
| Capability | `recon.network.portscan` |
| Binary | `nmap` (or absolute path) |

### Installation

```bash
# Debian/Ubuntu example
sudo apt-get install nmap
```

CYBEROS does not vendor nmap. The binary must be present on the host and **allowlisted** in process policy.

### Execution flow

```
Assessment / Task
  → Control Plane decision
  → Execution Grant
  → Tool Broker
  → Sandbox
  → Executor (tools.Manager registry)
  → nmap.Adapter
  → process.Runner (allowlist, no shell)
  → NormalizedResult → Evidence / Finding
```

### Process policy (required)

```go
process.NewRunner(process.Policy{
    Enabled: true,
    AllowedBins: []string{"nmap", "/usr/bin/nmap"},
    AllowShell: false, // must remain false
})
```

Default policy is **fail-closed** (no process execution).

### Parameters

| Key | Required | Description |
|-----|----------|-------------|
| `target` | yes | Host/IP/CIDR (no shell metacharacters) |
| `mode` | no | `list` (default, `-sL`) or `connect` (`-sT`) |
| `ports` | no | For connect mode, e.g. `80,443` |
| `timeout` | no | Duration string or `time.Duration` |

### Security restrictions

- No shell / no arbitrary command strings
- Argv only; target/ports validated against metacharacters
- No full host environment inheritance
- Timeout and output limits from process policy
- Adapter never issues grants
- Capability discovery ≠ authorization
- Unknown capability → rejected

### Adding future tools

1. Implement `tools.Adapter` in a separate package (e.g. `internal/tools/nuclei`).
2. Register with `tools.Manager.Register`.
3. Call `RegisterAllExecutors` into `execution.Registry`.
4. Do **not** modify Broker, Control Plane, or core packages.

Python/Rust tools use the same boundary via process allowlist + adapter, not as core dependencies.
