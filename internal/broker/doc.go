// Package broker implements ExecutionGrant issuance and the Tool Broker.
//
// Invariant:
//
//	NO VALID GRANT  ⇒  NO EXECUTION
//
// The Tool Broker is the sole execution gateway. Phase 1G only registers
// a safe mock executor — no shell, no real tools, no plugins.
package broker
