// Package recon defines the Recon tool pack metadata.
// Nmap is the reference member; future tools register the same way without
// modifying CYBEROS core (Control Plane, Broker, Sandbox, Scheduler, Worker).
package recon

import (
	"cyberos/internal/contracts"
	"cyberos/internal/tools"
	"cyberos/internal/tools/nmap"
)

// PackID is the recon pack identifier.
const PackID = tools.PackRecon

// Definition returns the recon pack with known tool IDs.
func Definition() tools.Pack {
	return tools.Pack{
		ID:          PackID,
		Name:        "Recon Pack",
		Description: "Network and host reconnaissance tools",
		ToolIDs:     []string{nmap.ToolID},
	}
}

// Register loads the Nmap adapter into the platform (if not present) and attaches the pack.
func Register(p *tools.Platform) error {
	if p == nil {
		return contracts.ErrInvalidArgument
	}
	if _, err := p.Resolve(nmap.ToolID); err != nil {
		if err := p.RegisterAdapter(nmap.New(nmap.Options{})); err != nil {
			return err
		}
	}
	return p.RegisterPack(Definition())
}
