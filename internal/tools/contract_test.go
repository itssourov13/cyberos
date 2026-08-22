package tools

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestPackContractVersionBackwardCompat(t *testing.T) {
	r := NewPackRegistry()
	// No ContractVersion — defaults on register
	if err := r.Register(Pack{ID: "pack.t", Name: "T", ToolIDs: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	p, _ := r.Get("pack.t")
	if p.ContractVersion != contracts.ExtensionContractV1 {
		t.Fatal(p.ContractVersion)
	}
}

func TestPackUnsupportedContractRejected(t *testing.T) {
	r := NewPackRegistry()
	err := r.Register(Pack{
		ID: "pack.bad", Name: "B", Version: "1.0.0",
		ContractVersion: "42", ToolIDs: []string{"a"},
	})
	if err == nil {
		t.Fatal("unsupported contract")
	}
}

func TestDescriptorPropagatesContractVersion(t *testing.T) {
	d := Descriptor{
		ID: "tool.a", Name: "A", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.a", ContractVersion: contracts.ExtensionContractV1,
	}
	m := d.ToolManifest()
	if m.ContractVersion != contracts.ExtensionContractV1 {
		t.Fatal(m)
	}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}
