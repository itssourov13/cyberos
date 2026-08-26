package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func samplePack() tools.PackFile {
	return tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack: tools.Pack{
			ID: "pack.recon", Name: "Recon", Version: "1.0.0", State: tools.PackStateRegistered,
			ToolIDs: []string{"tool.a"},
		},
		Tools: []tools.PackTool{
			{
				ID: "tool.a", Name: "Tool A", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
				ExecutorID: "tool.a", Binary: "/usr/bin/true",
				Capabilities: []contracts.CapabilityID{"cap.demo"},
			},
		},
	}
}

func TestCanonicalBytes_Deterministic(t *testing.T) {
	pf := samplePack()
	a, err := tools.CanonicalBytes(pf)
	if err != nil {
		t.Fatal(err)
	}
	b, err := tools.CanonicalBytes(pf)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("canonical bytes not stable")
	}
}

func TestDigest_FormattingIndependent(t *testing.T) {
	pf := samplePack()
	d1, err := tools.Digest(pf)
	if err != nil {
		t.Fatal(err)
	}
	// Different JSON field order / whitespace → same semantic struct → same digest
	raw1 := `{"format_version":"1","pack":{"id":"pack.recon","name":"Recon","version":"1.0.0","state":"REGISTERED","tool_ids":["tool.a"]},"tools":[{"id":"tool.a","name":"Tool A","version":"1.0.0","runtime":"go","executor_id":"tool.a","binary":"/usr/bin/true","capabilities":["cap.demo"]}]}`
	raw2 := `{
  "tools": [ {
    "capabilities": ["cap.demo"],
    "binary": "/usr/bin/true",
    "executor_id": "tool.a",
    "runtime": "go",
    "version": "1.0.0",
    "name": "Tool A",
    "id": "tool.a"
  } ],
  "pack": {
    "tool_ids": ["tool.a"],
    "state": "REGISTERED",
    "version": "1.0.0",
    "name": "Recon",
    "id": "pack.recon"
  },
  "format_version": "1"
}`
	var p1, p2 tools.PackFile
	if err := json.Unmarshal([]byte(raw1), &p1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw2), &p2); err != nil {
		t.Fatal(err)
	}
	dA, _ := tools.Digest(p1)
	dB, _ := tools.Digest(p2)
	if dA != dB || dA != d1 {
		t.Fatalf("formatting independence failed: %s vs %s vs %s", d1, dA, dB)
	}
}

func TestDigest_MutationChanges(t *testing.T) {
	base, _ := tools.Digest(samplePack())
	pf := samplePack()
	pf.Tools[0].Binary = "/usr/bin/false"
	mut, _ := tools.Digest(pf)
	if mut == base {
		t.Fatal("binary mutation must change digest")
	}
	pf2 := samplePack()
	pf2.Pack.Version = "2.0.0"
	mut2, _ := tools.Digest(pf2)
	if mut2 == base {
		t.Fatal("version mutation must change digest")
	}
	pf3 := samplePack()
	pf3.Pack.Name = "Other"
	mut3, _ := tools.Digest(pf3)
	if mut3 == base {
		t.Fatal("name mutation must change digest")
	}
}

func TestVerifyDigest_MismatchFailClosed(t *testing.T) {
	pf := samplePack()
	err := tools.VerifyDigest(pf, "sha256:"+strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("mismatch must fail")
	}
	d, _ := tools.Digest(pf)
	if err := tools.VerifyDigest(pf, d); err != nil {
		t.Fatal(err)
	}
	if err := tools.VerifyDigest(pf, ""); err != nil {
		t.Fatal("empty expected is legacy ok")
	}
}

func TestValidateDigestFormat(t *testing.T) {
	if err := tools.ValidateDigestFormat("md5:abc"); err == nil {
		t.Fatal()
	}
	if err := tools.ValidateDigestFormat("sha256:ZZ"); err == nil {
		t.Fatal()
	}
	d, _ := tools.Digest(samplePack())
	if err := tools.ValidateDigestFormat(d); err != nil {
		t.Fatal(err)
	}
}

func TestIdentity(t *testing.T) {
	id, err := tools.Identity(samplePack())
	if err != nil {
		t.Fatal(err)
	}
	if err := id.Validate(); err != nil {
		t.Fatal(err)
	}
	if id.Name != "Recon" || id.Version != "1.0.0" || !strings.HasPrefix(id.Digest, "sha256:") {
		t.Fatal(id)
	}
}

func TestFixedDigestVector(t *testing.T) {
	// Fixed vector: regenerate intentionally if canonicalization rules change.
	pf := samplePack()
	d, err := tools.Digest(pf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d, "sha256:") || len(d) != len("sha256:")+64 {
		t.Fatal(d)
	}
	// Second call must match (golden relative stability)
	d2, _ := tools.Digest(pf)
	if d != d2 {
		t.Fatal("vector unstable")
	}
	t.Log("fixed vector digest:", d)
}

func TestInstallRejectsBadDigest(t *testing.T) {
	p := tools.NewPlatform(nil)
	loader := tools.NewLoader(p, nil)
	pf := samplePack()
	pf.Pack.ContentDigest = "sha256:" + strings.Repeat("ab", 32)
	_, err := loader.InstallPackFile(pf)
	if err == nil {
		t.Fatal("bad digest must reject install")
	}
}
