# UniversalAIBridge (UAB) Integration

**Phase 0 Foundation Document — Future Design**

## Correct Integration Path

```
UAB
  ↓
CYBEROS SDK / API
  ↓
Tasks / Capabilities
  ↓
CONTROL PLANE
  ↓
EXECUTION GRANT
  ↓
TOOL BROKER
```

## Forbidden Paths

- UAB → direct shell
- UAB → direct database
- UAB → direct executor

## Rules

- UAB must remain replaceable.
- Every sensitive operation still passes through CYBEROS authorization.
- UAB is an external integration, not part of the CYBEROS core.

## Status

Not implemented in Phase 0 or Phase 1.
