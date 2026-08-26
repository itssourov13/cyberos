package tools_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func samplePackB() tools.PackFile {
	return tools.PackFile{
		FormatVersion: tools.PackFormatV1,
		Pack: tools.Pack{
			ID: "pack.recon", Name: "Recon", Version: "1.0.0", State: tools.PackStateRegistered,
			ToolIDs: []string{"tool.a"},
		},
		Tools: []tools.PackTool{{
			ID: "tool.a", Name: "Tool A", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
			ExecutorID: "tool.a", Binary: "/usr/bin/true",
			Capabilities: []contracts.CapabilityID{"cap.demo"},
		}},
	}
}

func TestSignAndVerify_Happy(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	reg := tools.NewSignerKeyRegistry()
	if err := reg.Register(tools.SignerKey{KeyID: "org.test/root", PublicKey: pub}); err != nil {
		t.Fatal(err)
	}
	signed, err := tools.SignPack(samplePackB(), "org.test/root", priv)
	if err != nil {
		t.Fatal(err)
	}
	res := tools.VerifyPackSignature(signed, reg)
	if !res.Valid || !res.Signed || res.KeyID != "org.test/root" {
		t.Fatalf("%+v", res)
	}
	if res.Digest == "" || res.Identity.Name != "Recon" {
		t.Fatal(res)
	}
}

func TestTamper_Fails(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub})
	signed, _ := tools.SignPack(samplePackB(), "k1", priv)
	signed.Tools[0].Binary = "/usr/bin/evil"
	res := tools.VerifyPackSignature(signed, reg)
	if res.Valid {
		t.Fatal("tamper accepted")
	}
}

func TestUnknownKey_Fails(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	signed, _ := tools.SignPack(samplePackB(), "attacker", priv)
	res := tools.VerifyPackSignature(signed, reg)
	if res.Valid || res.Reason != "UNKNOWN_KEY" {
		t.Fatalf("%+v", res)
	}
}

func TestDisabledKey_Fails(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub})
	signed, _ := tools.SignPack(samplePackB(), "k1", priv)
	_ = reg.Disable("k1")
	res := tools.VerifyPackSignature(signed, reg)
	if res.Valid || res.Reason != "KEY_DISABLED" {
		t.Fatalf("%+v", res)
	}
}

func TestDuplicateKeyID_Rejected(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub})
	err := reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub2})
	if err == nil {
		t.Fatal("duplicate key replace must fail")
	}
}

func TestWrongKey_Fails(t *testing.T) {
	pubA, privA, _ := ed25519.GenerateKey(nil)
	pubB, _, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "kA", PublicKey: pubA})
	_ = reg.Register(tools.SignerKey{KeyID: "kB", PublicKey: pubB})
	signed, _ := tools.SignPack(samplePackB(), "kA", privA)
	signed.Signature.KeyID = "kB"
	res := tools.VerifyPackSignature(signed, reg)
	if res.Valid {
		t.Fatal("wrong key")
	}
}

func TestMalformedSignature_Fails(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub})
	signed, _ := tools.SignPack(samplePackB(), "k1", priv)
	signed.Signature.Signature = base64.StdEncoding.EncodeToString([]byte("short"))
	res := tools.VerifyPackSignature(signed, reg)
	if res.Valid {
		t.Fatal()
	}
	signed.Signature.Signature = "!!!not-b64!!!"
	res = tools.VerifyPackSignature(signed, reg)
	if res.Valid {
		t.Fatal()
	}
}

func TestSignedRequired_Loader(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "k1", PublicKey: pub})
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	loader.SignerKeys = reg
	loader.VerifyMode = tools.VerifySignedRequired
	_, err := loader.InstallPackFile(samplePackB())
	if err == nil {
		t.Fatal("unsigned must fail SIGNED_REQUIRED")
	}
	signed, _ := tools.SignPack(samplePackB(), "k1", priv)
	_, err = loader.InstallPackFile(signed)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLegacyUnsigned_NotReportedSigned(t *testing.T) {
	res := tools.VerifyPack(samplePackB(), nil, tools.VerifyLegacy)
	if !res.Valid || res.Signed {
		t.Fatalf("%+v", res)
	}
}
