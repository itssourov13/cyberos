// Package contracts defines the stable, infrastructure-neutral contract layer
// for CYBEROS.
//
// This package is the top of the dependency hierarchy:
//
//	Contracts
//	    ↓
//	Domain
//	    ↓
//	Application
//	    ↓
//	Control Plane
//	    ↓
//	Ports
//	    ↓
//	Adapters
//
// Rules:
//   - Contracts MUST remain free of PostgreSQL, SQL, HTTP, CLI, filesystem,
//     AI-provider, tool-specific, or any infrastructure types.
//   - External consumers (SDK, future plugins, agents, ZOYA, UAB) must be able
//     to depend on these contracts without pulling in implementation details.
//   - The Control Plane decision is a structured object, never a bare bool.
//   - No valid ExecutionGrant means no execution (future invariant).
//
// Phase 1A establishes only the shapes and invariants. Implementations belong
// to later phases.
package contracts
