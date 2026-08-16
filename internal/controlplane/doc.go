// Package controlplane is the security authority of CYBEROS.
//
// It evaluates the non-bypassable chain:
//
//	Scope → Principal → Authorization → Policy → Permission → Risk → Approval
//
// and returns a structured Decision (never a bare boolean).
//
// Phase 1E implements evaluation and decision formation only.
// ExecutionGrant issuance and Tool Broker belong to later phases.
package controlplane
