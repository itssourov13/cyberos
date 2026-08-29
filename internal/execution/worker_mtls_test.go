package execution_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"cyberos/internal/execution"
	"cyberos/internal/storage/memory"
)

func fpOf(cert *x509.Certificate) string {
	s := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(s[:])
}

func testCert(t *testing.T) *x509.Certificate {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(99),
		Subject:      pkix.Name{CommonName: "w1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAuthenticateTLS_Match(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	cert := testCert(t)
	fp := fpOf(cert)
	_, err := reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", Status: execution.WorkerActive,
		CertFingerprint: fp, CredentialRef: "ref://1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.Activate(ctx, "proj-a", "w1")
	if err != nil {
		// may already active
		_ = err
	}
	w, err := reg.AuthenticateTLS(ctx, "proj-a", "w1", "h1", fp)
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != "w1" {
		t.Fatal()
	}
}

func TestAuthenticateTLS_WrongHost(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	cert := testCert(t)
	fp := fpOf(cert)
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", Status: execution.WorkerActive,
		CertFingerprint: fp, CredentialRef: "ref://1",
	})
	_, _ = reg.Activate(ctx, "proj-a", "w1")
	_, err := reg.AuthenticateTLS(ctx, "proj-a", "w1", "h2", fp)
	if err == nil {
		t.Fatal()
	}
}

func TestAuthenticateTLS_WrongFingerprint(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", Status: execution.WorkerActive,
		CertFingerprint: "deadbeef", CredentialRef: "ref://1",
	})
	_, _ = reg.Activate(ctx, "proj-a", "w1")
	_, err := reg.AuthenticateTLS(ctx, "proj-a", "w1", "h1", "cafebabe")
	if err == nil {
		t.Fatal()
	}
}

func TestAuthenticateTLS_CrossProject(t *testing.T) {
	reg := execution.NewWorkerRegistry(memory.NewWorkerRepo())
	ctx := context.Background()
	cert := testCert(t)
	fp := fpOf(cert)
	_, _ = reg.Register(ctx, execution.Worker{
		ID: "w1", ProjectID: "proj-a", HostID: "h1", Status: execution.WorkerActive,
		CertFingerprint: fp, CredentialRef: "ref://1",
	})
	_, err := reg.AuthenticateTLS(ctx, "proj-b", "w1", "h1", fp)
	if err == nil {
		t.Fatal("cross-project must fail")
	}
}
