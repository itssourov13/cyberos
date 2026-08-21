package process

import (
	"time"

	"cyberos/internal/config"
)

// PolicyFromConfig builds a process.Policy from application config.
// Defaults remain fail-closed (Enabled=false, AllowShell=false).
func PolicyFromConfig(c *config.Config) Policy {
	p := DefaultPolicy()
	if c == nil {
		return p
	}
	p.Enabled = c.ProcessEnabled
	p.AllowShell = c.ProcessAllowShell
	if len(c.ProcessAllowedBins) > 0 {
		p.AllowedBins = append([]string{}, c.ProcessAllowedBins...)
	}
	if len(c.ProcessAllowedDirs) > 0 {
		p.AllowedDirs = append([]string{}, c.ProcessAllowedDirs...)
	}
	if c.ProcessTimeout != "" {
		if d, err := time.ParseDuration(c.ProcessTimeout); err == nil && d > 0 {
			p.DefaultTimeout = d
		}
	}
	if c.ProcessMaxOutput > 0 {
		p.MaxOutputBytes = c.ProcessMaxOutput
	}
	// Hard safety: never enable shell unless explicitly allowlisted and enabled.
	if !p.Enabled {
		p.AllowShell = false
	}
	return p
}
