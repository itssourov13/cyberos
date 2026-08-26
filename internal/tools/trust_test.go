package tools_test

import (
	"crypto/ed25519"
	"testing"

	"cyberos/internal/tools"
)

func signedSample(t *testing.T) (tools.PackFile, *tools.SignerKeyRegistry, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	reg := tools.NewSignerKeyRegistry()
	_ = reg.Register(tools.SignerKey{KeyID: "org.test/root", PublicKey: pub})
	signed, err := tools.SignPack(samplePackB(), "org.test/root", priv)
	if err != nil {
		t.Fatal(err)
	}
	return signed, reg, priv
}

func TestTrust_VerifiedNotTrusted(t *testing.T) {
	signed, reg, _ := signedSample(t)
	ev := &tools.TrustEvaluator{Keys: reg, Store: tools.NewTrustStore(), Policy: tools.NewTrustPolicy()}
	d := ev.Evaluate("proj-a", signed)
	if d.State != tools.TrustVerified {
		t.Fatalf("%+v", d)
	}
}

func TestTrust_ExplicitDigest(t *testing.T) {
	signed, reg, _ := signedSample(t)
	store := tools.NewTrustStore()
	d0, _ := tools.Digest(signed)
	// digest excludes signature meta — use verification digest
	vres := tools.VerifyPackSignature(signed, reg)
	_ = store.TrustPack("proj-a", vres.Digest)
	ev := &tools.TrustEvaluator{Keys: reg, Store: store, Policy: tools.NewTrustPolicy()}
	d := ev.Evaluate("proj-a", signed)
	if d.State != tools.TrustTrusted {
		t.Fatalf("%+v digest=%s computed=%s", d, vres.Digest, d0)
	}
}

func TestTrust_PackRevoked(t *testing.T) {
	signed, reg, _ := signedSample(t)
	store := tools.NewTrustStore()
	vres := tools.VerifyPackSignature(signed, reg)
	_ = store.TrustPack("proj-a", vres.Digest)
	_ = store.RevokePack("proj-a", vres.Digest, "vuln", "op")
	ev := &tools.TrustEvaluator{Keys: reg, Store: store}
	d := ev.Evaluate("proj-a", signed)
	if d.State != tools.TrustRevoked || d.Reason != "PACK_REVOKED" {
		t.Fatalf("%+v", d)
	}
}

func TestTrust_SignerRevoked(t *testing.T) {
	signed, reg, _ := signedSample(t)
	store := tools.NewTrustStore()
	vres := tools.VerifyPackSignature(signed, reg)
	_ = store.TrustPack("proj-a", vres.Digest)
	_ = reg.Revoke("org.test/root")
	ev := &tools.TrustEvaluator{Keys: reg, Store: store}
	d := ev.Evaluate("proj-a", signed)
	if d.State != tools.TrustRevoked {
		t.Fatalf("%+v", d)
	}
}

func TestTrust_CrossProject(t *testing.T) {
	signed, reg, _ := signedSample(t)
	store := tools.NewTrustStore()
	vres := tools.VerifyPackSignature(signed, reg)
	_ = store.TrustPack("proj-a", vres.Digest)
	ev := &tools.TrustEvaluator{Keys: reg, Store: store}
	d := ev.Evaluate("proj-b", signed)
	if d.State == tools.TrustTrusted {
		t.Fatal("cross-project trust leak")
	}
}

func TestTrust_CannotTrustRevoked(t *testing.T) {
	store := tools.NewTrustStore()
	dig := "sha256:" + "ab" + string(make([]byte, 62))
	// use valid hex
	dig = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_ = store.RevokePack("p", dig, "x", "y")
	err := store.TrustPack("p", dig)
	if err == nil {
		t.Fatal()
	}
}

func TestTrustRequired_Loader(t *testing.T) {
	signed, reg, _ := signedSample(t)
	store := tools.NewTrustStore()
	plat := tools.NewPlatform(nil)
	loader := tools.NewLoader(plat, nil)
	loader.SignerKeys = reg
	loader.VerifyMode = tools.VerifyTrustRequired
	loader.TrustEval = &tools.TrustEvaluator{Keys: reg, Store: store}
	loader.ProjectID = "proj-a"
	_, err := loader.InstallPackFile(signed)
	if err == nil {
		t.Fatal("verified but not trusted must fail TRUST_REQUIRED")
	}
	vres := tools.VerifyPackSignature(signed, reg)
	_ = store.TrustPack("proj-a", vres.Digest)
	_, err = loader.InstallPackFile(signed)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSignerInheritance_DefaultOff(t *testing.T) {
	signed, reg, _ := signedSample(t)
	pol := tools.NewTrustPolicy()
	pol.TrustedKeyIDs["org.test/root"] = struct{}{}
	// inheritance off
	ev := &tools.TrustEvaluator{Keys: reg, Store: tools.NewTrustStore(), Policy: pol}
	d := ev.Evaluate("proj-a", signed)
	if d.State == tools.TrustTrusted {
		t.Fatal("inheritance must default off")
	}
	pol.AllowSignerInheritance = true
	d = ev.Evaluate("proj-a", signed)
	if d.State != tools.TrustTrusted {
		t.Fatal(d)
	}
}
