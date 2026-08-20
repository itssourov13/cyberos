package tools

import "cyberos/internal/contracts"

// RuntimeLabels documents supported adapter runtime labels.
// Execution still goes through process policy / Broker; labels are metadata.
func RuntimeLabels() []contracts.PluginRuntime {
	return []contracts.PluginRuntime{
		contracts.PluginRuntimeGo,
		contracts.PluginRuntimePython,
		contracts.PluginRuntimeRust,
		contracts.PluginRuntimeOther,
	}
}
