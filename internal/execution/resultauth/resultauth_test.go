package resultauth_test

import (
	"crypto/ed25519"
	"testing"

	"cyberos/internal/execution/resultauth"
)

func TestSignVerify_OK(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	e := resultauth.Envelope{
		Version: "1", AssignmentID: "a1", GrantID: "g1", ProjectID: "p1",
		WorkerID: "w1", HostID: "h1", PackContentDigest: "sha256:aa",
		ResultHash: resultauth.HashPayload("SUCCEEDED", "out"), ResultStatus: "SUCCEEDED",
		ResultID: "r1", CompletedAt: "2026-01-01T00:00:00Z",
	}
	signed, err := resultauth.Sign(e, priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := resultauth.Verify(signed, pub); err != nil {
		t.Fatal(err)
	}
}

func TestVerify_Tamper(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	e, _ := resultauth.Sign(resultauth.Envelope{
		AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1",
		ResultHash: "hh", ResultStatus: "SUCCEEDED", ResultID: "r1", CompletedAt: "2026-01-01T00:00:00Z",
	}, priv)
	e.ResultStatus = "FAILED"
	if err := resultauth.Verify(e, pub); err == nil {
		t.Fatal()
	}
}

func TestVerifyCompletion_Bindings(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	keys := resultauth.NewKeyRegistry()
	_ = keys.Register("w1", pub)
	bind := resultauth.AssignmentBinding{
		AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1",
		PackContentDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	e, _ := resultauth.Sign(resultauth.Envelope{
		AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1",
		PackContentDigest: bind.PackContentDigest, ResultHash: "hh", ResultStatus: "SUCCEEDED",
		ResultID: "r1", CompletedAt: "2026-01-01T00:00:00Z",
	}, priv)
	if err := resultauth.VerifyCompletion(e, bind, keys); err != nil {
		t.Fatal(err)
	}
	e2 := e
	e2.PackContentDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	e2, _ = resultauth.Sign(e2, priv)
	if err := resultauth.VerifyCompletion(e2, bind, keys); err == nil {
		t.Fatal("pack mismatch")
	}
}

func TestReplayGuard(t *testing.T) {
	g := resultauth.NewReplayGuard()
	_ = g.Accept("r1")
	if err := g.Accept("r1"); err == nil {
		t.Fatal()
	}
}

func TestDisabledKey(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	keys := resultauth.NewKeyRegistry()
	_ = keys.Register("w1", pub)
	keys.Disable("w1")
	e, _ := resultauth.Sign(resultauth.Envelope{
		AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1",
		ResultHash: "h", ResultStatus: "SUCCEEDED", ResultID: "r1", CompletedAt: "2026-01-01T00:00:00Z",
	}, priv)
	bind := resultauth.AssignmentBinding{AssignmentID: "a1", GrantID: "g1", ProjectID: "p1", WorkerID: "w1", HostID: "h1"}
	if err := resultauth.VerifyCompletion(e, bind, keys); err == nil {
		t.Fatal()
	}
}
