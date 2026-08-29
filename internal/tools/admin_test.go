package tools_test

import (
	"crypto/ed25519"
	"testing"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/events"
	"cyberos/internal/tools"
)

func TestTrustAdmin_HumanDenied(t *testing.T) {
	a := &tools.TrustAdmin{Store: tools.NewTrustStore(), Keys: tools.NewSignerKeyRegistry()}
	err := a.TrustPack(contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman}, "proj-a",
		"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "c1", "x")
	if err == nil {
		t.Fatal("human must be denied")
	}
}

func TestTrustAdmin_AgentDenied(t *testing.T) {
	a := &tools.TrustAdmin{Store: tools.NewTrustStore(), Keys: tools.NewSignerKeyRegistry()}
	err := a.TrustPack(contracts.PrincipalRef{ID: "agent-1", Type: contracts.PrincipalTypeAgent}, "proj-a",
		"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "c1", "x")
	if err == nil {
		t.Fatal("agent must be denied")
	}
}

func TestTrustAdmin_ServiceAllowed_Audited(t *testing.T) {
	store := tools.NewTrustStore()
	aud := audit.NewService(events.NewStore())
	a := &tools.TrustAdmin{Store: store, Keys: tools.NewSignerKeyRegistry(), Audit: aud}
	op := contracts.PrincipalRef{ID: "svc-ops", Type: contracts.PrincipalTypeService}
	dig := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if err := a.TrustPack(op, "proj-a", dig, "corr-1", "approved"); err != nil {
		t.Fatal(err)
	}
	if !a.IsTrusted("proj-a", dig) {
		t.Fatal()
	}
	if a.IsTrusted("proj-b", dig) {
		t.Fatal("cross-project")
	}
	if aud.Len() < 1 {
		t.Fatal("audit required")
	}
	if err := a.RevokePack(op, "proj-a", dig, "vuln", "corr-2"); err != nil {
		t.Fatal(err)
	}
}

func TestTrustAdmin_RegisterSigner(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	a := &tools.TrustAdmin{Store: tools.NewTrustStore(), Keys: tools.NewSignerKeyRegistry()}
	op := contracts.PrincipalRef{ID: "svc-ops", Type: contracts.PrincipalTypeService}
	if err := a.RegisterSigner(op, tools.SignerKey{KeyID: "k-ops", PublicKey: pub}, "c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetSigner("k-ops"); err != nil {
		t.Fatal(err)
	}
	// human cannot
	if err := a.DisableSigner(contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman}, "k-ops", "c2"); err == nil {
		t.Fatal()
	}
}
