package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
)

func signedFixture(t *testing.T) (*Issuer, *Broker, contracts.ExecutionGrant) {
	t.Helper()
	signer, err := NewEphemeralGrantSigner()
	if err != nil {
		t.Fatal(err)
	}
	iss := NewIssuer(time.Minute)
	iss.Signer = signer
	brk := New(mock.New())
	brk.Signer = signer

	g, err := iss.Issue(context.Background(), IssueRequest{
		Decision: contracts.Decision{
			ID:           "dec-1",
			Timestamp:    time.Now().UTC(),
			Outcome:      contracts.DecisionAllowed,
			Principal:    contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
			ProjectID:    "proj",
			ScopeID:      "scope",
			TaskID:       "task",
			CapabilityID: "cap",
			Reason:       "ok",
		},
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	return iss, brk, g.ExecutionGrant
}

func execReq(g contracts.ExecutionGrant) contracts.ExecutionRequest {
	return contracts.ExecutionRequest{Grant: g}
}

func TestIssuedGrantIsSignedAndAccepted(t *testing.T) {
	_, brk, g := signedFixture(t)
	if g.Signature == "" {
		t.Fatal("issued grant must be signed")
	}
	if _, err := brk.Execute(execReq(g)); err != nil {
		t.Fatalf("legitimately issued grant rejected: %v", err)
	}
}

func TestForgedUnsignedGrantRejected(t *testing.T) {
	_, brk, g := signedFixture(t)
	g.Signature = ""
	g.Nonce = "forged-nonce"
	res, err := brk.Execute(execReq(g))
	if !errors.Is(err, contracts.ErrGrantInvalid) || res.Status != contracts.ExecutionStatusRejected {
		t.Fatalf("unsigned grant must be rejected, err=%v status=%s", err, res.Status)
	}
}

func TestTamperedGrantRejected(t *testing.T) {
	cases := map[string]func(*contracts.ExecutionGrant){
		"principal":  func(g *contracts.ExecutionGrant) { g.Principal.ID = "attacker" },
		"scope":      func(g *contracts.ExecutionGrant) { g.ScopeID = "other" },
		"capability": func(g *contracts.ExecutionGrant) { g.CapabilityID = "other" },
		"expiry":     func(g *contracts.ExecutionGrant) { g.ExpiresAt = g.ExpiresAt.Add(24 * time.Hour) },
		"pack": func(g *contracts.ExecutionGrant) {
			g.PackContentDigest = "sha256:" + string(make([]byte, 0)) + "0000000000000000000000000000000000000000000000000000000000000000"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, brk, g := signedFixture(t)
			mutate(&g)
			if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
				t.Fatalf("tampered %s must be rejected, got %v", name, err)
			}
		})
	}
}

func TestGrantSignedByForeignKeyRejected(t *testing.T) {
	_, brk, _ := signedFixture(t)
	_, _, foreign := signedFixture(t) // different ephemeral key
	if _, err := brk.Execute(execReq(foreign)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("grant from another issuer must be rejected, got %v", err)
	}
}

func TestRevokedGrantRejected(t *testing.T) {
	_, brk, g := signedFixture(t)
	brk.RevokeGrant(g.ID)
	if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("revoked grant must be rejected, got %v", err)
	}
}

func TestExpiredAndReplayedSignedGrantRejected(t *testing.T) {
	_, brk, g := signedFixture(t)
	if _, err := brk.Execute(execReq(g)); err != nil {
		t.Fatal(err)
	}
	if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("replay must be rejected, got %v", err)
	}

	_, brk2, g2 := signedFixture(t)
	brk2.Now = func() time.Time { return g2.ExpiresAt.Add(time.Second) }
	if _, err := brk2.Execute(execReq(g2)); !errors.Is(err, contracts.ErrGrantExpired) {
		t.Fatalf("expired grant must be rejected, got %v", err)
	}
}

func TestShortSigningKeyRefused(t *testing.T) {
	if _, err := NewGrantSigner([]byte("short")); err == nil {
		t.Fatal("short key must be refused")
	}
}
