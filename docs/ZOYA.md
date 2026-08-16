# ZOYA / JARVIS Architecture

**Phase 0 Foundation Document — Future Design**

## Relationship

```
ZOYA / JARVIS
      ↓
AI Orchestrator
      ↓
CYBEROS SDK / API
      ↓
CONTROL PLANE
      ↓
TASK ENGINE
      ↓
TOOL BROKER
```

**Correct dependency direction:** ZOYA depends on CYBEROS.  
CYBEROS must **never** depend on ZOYA.

## Intended Capabilities (Future)

- Understand natural-language security objectives
- Plan workflows and decompose tasks
- Coordinate agents
- Discover capabilities
- Analyze evidence and correlate findings
- Assist with reports
- Observe events
- Request human approvals
- Drive long-running workflows

## Authority

ZOYA is **not** the security authority.  
CYBEROS Control Plane remains the sole authority for authorization and execution grants.

## Status

Not implemented in Phase 0 or Phase 1.
