// Package version provides build and version information for CYBEROS.
// Phase 0 Foundation.
package version

import "fmt"

// These can be overridden at build time via -ldflags.
var (
	Version   = "0.0.0-foundation"
	Commit    = "unknown"
	BuildDate = "unknown"
	GoVersion = "go1.22.2"
)

// String returns a human-readable version string.
func String() string {
	return fmt.Sprintf("CYBEROS %s (commit: %s, built: %s, go: %s)",
		Version, Commit, BuildDate, GoVersion)
}

// Info returns structured version data.
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     Commit,
		"build_date": BuildDate,
		"go_version": GoVersion,
	}
}
