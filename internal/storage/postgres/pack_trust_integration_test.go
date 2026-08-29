//go:build integration

package postgres_test

import (
	"crypto/ed25519"
	"os"
	"testing"

	"cyberos/internal/storage/postgres"
	"cyberos/internal/tools"
)

func dsnOrSkip(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		t.Skip("LIVE PG: SKIPPED — DSN unavailable")
	}
	return dsn
}

func TestPackTrust_Persistence(t *testing.T) {
	dsn := dsnOrSkip(t)
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := postgres.NewPackTrustRepo(db)
	dig := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := repo.TrustPack("proj-a", dig); err != nil {
		t.Fatal(err)
	}
	if !repo.IsTrusted("proj-a", dig) {
		t.Fatal("expected trusted")
	}
	if repo.IsTrusted("proj-b", dig) {
		t.Fatal("cross-project leak")
	}
	if err := repo.RevokePack("proj-a", dig, "vuln", "op"); err != nil {
		t.Fatal(err)
	}
	if !repo.IsRevoked("proj-a", dig) {
		t.Fatal("expected revoked")
	}
	if err := repo.TrustPack("proj-a", dig); err == nil {
		t.Fatal("cannot re-trust revoked")
	}
}

func TestSignerKey_Persistence(t *testing.T) {
	dsn := dsnOrSkip(t)
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := postgres.NewSignerKeyRepo(db)
	pub, _, _ := ed25519.GenerateKey(nil)
	id := "test-key-19b-persist"
	_ = repo.Register(tools.SignerKey{KeyID: id, PublicKey: pub})
	k, err := repo.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(k.PublicKey) != 32 {
		t.Fatal()
	}
	pub2, _, _ := ed25519.GenerateKey(nil)
	if err := repo.Register(tools.SignerKey{KeyID: id, PublicKey: pub2}); err == nil {
		t.Fatal("duplicate key_id must fail")
	}
}
