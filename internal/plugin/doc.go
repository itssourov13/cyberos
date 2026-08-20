// Package plugin provides plugin/adapter contracts and lifecycle management.
//
// Discovery and Prepare never authorize or execute tools.
// Execution must go through Tool Broker → Sandbox under a valid grant.
package plugin
