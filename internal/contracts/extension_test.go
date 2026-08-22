package contracts

import "testing"

func TestValidateExtensionContractV1(t *testing.T) {
	if err := ValidateExtensionContract(""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExtensionContract(ExtensionContractV1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExtensionContract("99"); err == nil {
		t.Fatal("unsupported must fail")
	}
	if NormalizeExtensionContract("") != ExtensionContractV1 {
		t.Fatal("normalize")
	}
}

func TestToolManifestContractVersion(t *testing.T) {
	m := ToolManifest{
		ID: "tool.x", Name: "X", Version: "1.0.0", Runtime: PluginRuntimeGo, ExecutorID: "tool.x",
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	m.ContractVersion = "999"
	if err := m.Validate(); err == nil {
		t.Fatal("bad contract version")
	}
	m.ContractVersion = ExtensionContractV1
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestToolManifestStillRequiresCoreFields(t *testing.T) {
	m := ToolManifest{ContractVersion: ExtensionContractV1}
	if err := m.Validate(); err == nil {
		t.Fatal("missing id")
	}
}
