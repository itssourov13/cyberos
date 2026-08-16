# Phase 1 — Secure Core

**Status:** Not started (awaiting approval)

## Strict Build Order

1. Contracts
2. Domain entities
3. Repository interfaces
4. PostgreSQL adapter
5. Identity / Principal model
6. Scope model
7. Authorization
8. Policy
9. Permission
10. Risk
11. Approval
12. Execution Grant
13. Control Plane
14. Capability Registry
15. Task state machine
16. Tool Broker interface
17. Safe mock executor
18. Evidence
19. Findings
20. Provenance
21. Events
22. Audit
23. API v1
24. CLI
25. SDK foundation
26. Integration tests
27. Security tests

## Hard Constraints for Phase 1

- Only safe mock execution is allowed
- No unrestricted shell
- No real pentesting tools
- No autonomous AI execution
- No Web UI
- No plugins / agents / ZOYA / UAB yet

## Success Criteria

- Control Plane can evaluate a request and produce a structured decision
- Execution Grant is issued only after the full chain
- Tool Broker accepts only valid grants and routes to mock executor
- Evidence / Findings / Provenance / Events / Audit are recorded
- Security tests protect the invariant
- CLI and API v1 expose the core operations through the same services
