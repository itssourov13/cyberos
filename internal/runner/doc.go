// Package runner is the durable task-runner foundation.
//
// PromoteDueScheduled and ClaimPending never authorize or execute work.
// Authorization remains Control Plane → Grant → Tool Broker → Sandbox.
package runner
