// Package scheduler provides task scheduling primitives.
//
// Invariant: Scheduling ≠ Authorization.
// A scheduled or queued task still requires the full Control Plane → Grant →
// Tool Broker chain before any execution.
package scheduler
