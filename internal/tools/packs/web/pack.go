// Package web defines the Web tool pack.
package web

import (
	"cyberos/internal/contracts"
	"cyberos/internal/tools"
	"cyberos/internal/tools/httpprobe"
)

const PackID = tools.PackWeb

// Definition returns the web pack.
func Definition() tools.Pack {
	return tools.Pack{
		ID: PackID, Name: "Web Pack", Description: "HTTP and web assessment tools",
		ToolIDs: []string{httpprobe.ToolID},
	}
}

// Register registers HTTP probe and the web pack.
func Register(p *tools.Platform) error {
	if p == nil {
		return contracts.ErrInvalidArgument
	}
	if _, err := p.Resolve(httpprobe.ToolID); err != nil {
		if err := p.RegisterAdapter(httpprobe.New(httpprobe.Options{})); err != nil {
			return err
		}
	}
	return p.RegisterPack(Definition())
}
