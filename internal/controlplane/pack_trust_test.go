package controlplane

import (
	"context"
	"testing"

	"cyberos/internal/application/scope"
	"cyberos/internal/identity"
)

func setupCPWithPackTrust(t *testing.T) (*Service, *memPrincipalRepo, *memScopeRepo) {
	t.Helper()
	pr := &memPrincipalRepo{}
	sr := &memScopeRepo{}
	idSvc := identity.NewService(pr)
	scSvc := scope.NewService(sr)
	cp := NewService(idSvc, scSvc, WithPolicyEvaluator(PackTrustPolicyEvaluator{Inner: DefaultPolicyEvaluator{}}))
	seedActive(t, pr, sr)
	return cp, pr, sr
}

func TestPackTrust_TrustedEligible(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, err := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED", PackContentDigest: "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed() {
		t.Fatalf("expected allow got %s", d.Reason)
	}
}

func TestPackTrust_VerifiedDenied(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, err := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: true, EffectivePackTrust: "VERIFIED",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed() {
		t.Fatal("VERIFIED must not satisfy RequireTrustedPack")
	}
	if d.Policy == nil || d.Policy.Reason != "PACK_NOT_TRUSTED" {
		t.Fatalf("%+v", d.Policy)
	}
}

func TestPackTrust_RevokedDenied(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: true, EffectivePackTrust: "REVOKED",
	})
	if d.Allowed() {
		t.Fatal()
	}
	if d.Policy.Reason != "PACK_REVOKED" {
		t.Fatal(d.Policy.Reason)
	}
}

func TestPackTrust_UntrustedDenied(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: true, EffectivePackTrust: "UNTRUSTED",
	})
	if d.Allowed() {
		t.Fatal()
	}
}

func TestPackTrust_NotRequired_Allows(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: false, EffectivePackTrust: "UNTRUSTED",
	})
	if !d.Allowed() {
		t.Fatal(d.Reason)
	}
}

func TestPackTrust_TrustedButCapabilityMissing_StillDenied(t *testing.T) {
	cp, _, _ := setupCPWithPackTrust(t)
	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1", CapabilityID: "",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED",
	})
	if d.Allowed() {
		t.Fatal("TRUSTED must not bypass capability policy")
	}
}

func TestPackTrust_CrossProjectDoesNotGrantEligibility(t *testing.T) {
	// Trust is supplied on the request by caller after project-scoped evaluation;
	// CP only sees EffectivePackTrust for *this* request's project.
	cp, _, _ := setupCPWithPackTrust(t)
	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-other", ScopeID: "scope-1", CapabilityID: "cap.demo",
		RequireTrustedPack: true, EffectivePackTrust: "UNTRUSTED", // B did not trust
	})
	if d.Allowed() {
		t.Fatal("cross-project untrusted must deny")
	}
}
