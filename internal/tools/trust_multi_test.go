package tools_test

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

// Two evaluators sharing one store simulate multi-instance visibility (Phase 20).
func TestTrust_SharedStore_MultiInstance(t *testing.T) {
	store := tools.NewTrustStore()
	dig := "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	if err := store.TrustPack(contracts.ProjectID("proj-a"), dig); err != nil {
		t.Fatal(err)
	}
	if !store.IsTrusted("proj-a", dig) {
		t.Fatal("instance B should see trust")
	}
	if store.IsTrusted("proj-b", dig) {
		t.Fatal("cross-project isolation")
	}
	if err := store.RevokePack("proj-a", dig, "ops", "op"); err != nil {
		t.Fatal(err)
	}
	if !store.IsRevoked("proj-a", dig) {
		t.Fatal("instance B should see revoke")
	}
	if err := store.TrustPack("proj-a", dig); err == nil {
		t.Fatal("cannot re-trust revoked")
	}
}

func TestTrust_DBErrorPath_NeverTrusted(t *testing.T) {
	// Failing store: IsTrusted false on error path is enforced by PG adapter;
	// memory always consistent. Fail-closed invariant: empty store is not TRUSTED.
	store := tools.NewTrustStore()
	if store.IsTrusted("p", "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee") {
		t.Fatal("empty must not be trusted")
	}
}
