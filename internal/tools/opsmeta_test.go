package tools_test

import (
	"strings"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func TestRuntimeMetaValid(t *testing.T) {
	r := tools.RuntimeMeta{Type: contracts.PluginRuntimeOther, Binary: "nmap", VersionConstraint: ">=7"}
	// PluginRuntimeOther may not exist - check Valid runtimes
	
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Binary = "nmap;rm"
	if err := r.Validate(); err == nil {
		t.Fatal("unsafe binary")
	}
}

func TestDependencyMissingFailClosed(t *testing.T) {
	plat := tools.NewPlatform(nil)
	err := tools.CheckDependencies(plat, []tools.DependencyMeta{{PackID: "pack.missing"}})
	if err == nil {
		t.Fatal("missing dep")
	}
}

func TestDependencyVersionConflict(t *testing.T) {
	plat := tools.NewPlatform(nil)
	_ = plat.RegisterPack(tools.Pack{ID: "pack.base", Name: "B", Version: "1.0.0"})
	err := tools.CheckDependencies(plat, []tools.DependencyMeta{{PackID: "pack.base", Version: "2.0.0"}})
	if err == nil {
		t.Fatal("version conflict")
	}
}

func TestConfigRefRejectsTraversal(t *testing.T) {
	c := tools.ConfigRef{Name: "../etc/passwd"}
	if err := c.Validate(); err == nil {
		t.Fatal("traversal")
	}
	c = tools.ConfigRef{Name: "scan.timeout"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSecretRefNoMaterial(t *testing.T) {
	s := tools.SecretRefMeta{Name: "api", Ref: "env:CYBEROS_API_TOKEN"}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Ref = "plaintext-secret"
	if err := s.Validate(); err == nil {
		t.Fatal("plaintext")
	}
}

func TestResourceHintsBounded(t *testing.T) {
	r := tools.ResourceHints{TimeoutSeconds: 99999}
	if err := r.Validate(); err == nil {
		t.Fatal("timeout")
	}
	r = tools.ResourceHints{TimeoutSeconds: 30, MaxOutputBytes: 1024}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPackFileOpsMetadataInstall(t *testing.T) {
	plat := tools.NewPlatform(nil)
	_ = plat.RegisterPack(tools.Pack{ID: "pack.base", Name: "Base", Version: "1.0.0", State: tools.PackStateActive})
	loader := tools.NewLoader(plat, nil)
	pf := tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack: tools.Pack{
			ID: "pack.ext", Name: "Ext", Version: "1.0.0", State: tools.PackStateActive,
			Ops: &tools.OpsMeta{
				Runtime:      &tools.RuntimeMeta{Type: contracts.PluginRuntimeGo, Binary: "tool"},
				Dependencies: []tools.DependencyMeta{{PackID: "pack.base", Version: "1.0.0"}},
				ConfigRefs:   []tools.ConfigRef{{Name: "tool.mode"}},
				SecretRefs:   []tools.SecretRefMeta{{Name: "tok", Ref: "env:CYBEROS_API_TOKEN"}},
				Resources:    &tools.ResourceHints{TimeoutSeconds: 60},
			},
		},
		Tools: []tools.PackTool{{
			ID: "tool.ext", Name: "Ext", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo, ExecutorID: "tool.ext",
		}},
	}
	if _, err := loader.InstallPackFile(pf); err != nil {
		t.Fatal(err)
	}
}

func TestOpsDoesNotAppearAsSecretInError(t *testing.T) {
	s := tools.SecretRefMeta{Name: "x", Ref: "env:FOO"}
	err := s.Validate()
	if err != nil {
		t.Fatal(err)
	}
	// invalid
	err = tools.SecretRefMeta{Name: "x", Ref: "not-a-ref"}.Validate()
	if err == nil {
		t.Fatal("expected err")
	}
	if strings.Contains(err.Error(), "password=") {
		t.Fatal("should not leak")
	}
}
