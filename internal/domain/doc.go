// Package domain contains the pure CYBEROS domain model.
//
// Dependency rules (strict):
//
//	internal/contracts  →  domain  →  (later layers)
//
// Domain may import only:
//   - Go standard library
//   - internal/contracts
//
// Domain must NOT import:
//   - storage / PostgreSQL / SQL
//   - HTTP / CLI / filesystem adapters
//   - control plane implementations
//   - tool brokers / executors
//   - AI, plugins, ZOYA, UAB, or external services
//
// Phase 1B implements entities, invariants, and the Task state machine.
// Control Plane, persistence, and execution belong to later phases.
package domain
