// Package taskengine implements the Task Engine.
//
// Responsibilities:
//   - Create tasks bound to Project, Scope, Principal, and Capability
//   - Enforce the explicit task state machine
//   - Reject illegal state transitions
//
// Non-responsibilities (later phases):
//   - Authorization decisions (Control Plane)
//   - ExecutionGrant issuance
//   - Tool Broker / execution
package taskengine
