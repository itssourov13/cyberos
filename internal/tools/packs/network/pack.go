// Package network defines the Network tool pack (runtime probes for multi-lang proof).
package network

import (
	"cyberos/internal/contracts"
	"cyberos/internal/tools"
	"cyberos/internal/tools/pythonprobe"
	"cyberos/internal/tools/rustprobe"
)

const PackID = tools.PackNet

// Definition returns the network pack including multi-runtime probes.
func Definition() tools.Pack {
	return tools.Pack{
		ID: PackID, Name: "Network Pack",
		Description: "Network/runtime tooling (Python/Rust probes as multi-runtime proof)",
		ToolIDs:     []string{pythonprobe.ToolID, rustprobe.ToolID},
	}
}

// Register registers Python/Rust probes and the network pack.
func Register(p *tools.Platform) error {
	if p == nil {
		return contracts.ErrInvalidArgument
	}
	for _, a := range []tools.Adapter{
		pythonprobe.New(pythonprobe.Options{}),
		rustprobe.New(rustprobe.Options{}),
	} {
		id := a.Descriptor().ID
		if _, err := p.Resolve(id); err != nil {
			if err := p.RegisterAdapter(a); err != nil {
				return err
			}
		}
	}
	return p.RegisterPack(Definition())
}
