package mtls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"cyberos/internal/execution/mtls"
)

func genCA(t *testing.T) (caCert *x509.Certificate, caKey *ecdsa.PrivateKey, caPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "CYBEROS-TEST-CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	b := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return cert, key, b
}

func genLeaf(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, cn string, notBefore, notAfter time.Time) (certPEM, keyPEM []byte, cert *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{cn, "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err = x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
	return certPEM, keyPEM, cert
}

func TestServerTLSConfig_RequiresClientCert(t *testing.T) {
	ca, caKey, caPEM := genCA(t)
	srvPEM, srvKey, _ := genLeaf(t, ca, caKey, "controller", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	cfg, err := mtls.ServerTLSConfig(mtls.Config{CACertPEM: caPEM, CertPEM: srvPEM, KeyPEM: srvKey})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal(cfg.ClientAuth)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("must not skip verify")
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatal(cfg.MinVersion)
	}
}

func TestClientTLSConfig_RequiresServerName(t *testing.T) {
	ca, caKey, caPEM := genCA(t)
	cliPEM, cliKey, _ := genLeaf(t, ca, caKey, "worker-1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	_, err := mtls.ClientTLSConfig(mtls.Config{CACertPEM: caPEM, CertPEM: cliPEM, KeyPEM: cliKey})
	if err == nil {
		t.Fatal("ServerName required")
	}
	cfg, err := mtls.ClientTLSConfig(mtls.Config{CACertPEM: caPEM, CertPEM: cliPEM, KeyPEM: cliKey, ServerName: "controller"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal()
	}
}

func TestBind_Match(t *testing.T) {
	ca, caKey, _ := genCA(t)
	_, _, cert := genLeaf(t, ca, caKey, "w1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	fp := mtls.FingerprintSHA256(cert)
	peer := mtls.PeerIdentity{Fingerprint: fp}
	err := mtls.Bind(peer, mtls.BindRequest{
		ProjectID: "proj-a", WorkerID: "w1", HostID: "h1", Fingerprint: fp, Status: "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBind_MismatchFingerprint(t *testing.T) {
	err := mtls.Bind(mtls.PeerIdentity{Fingerprint: "aaa"}, mtls.BindRequest{
		ProjectID: "p", WorkerID: "w", HostID: "h", Fingerprint: "bbb", Status: "ACTIVE",
	})
	if err == nil {
		t.Fatal()
	}
}

func TestBind_Revoked(t *testing.T) {
	err := mtls.Bind(mtls.PeerIdentity{Fingerprint: "aaa"}, mtls.BindRequest{
		ProjectID: "p", WorkerID: "w", HostID: "h", Fingerprint: "aaa", Status: "REVOKED",
	})
	if err == nil {
		t.Fatal()
	}
}

func TestBind_Disabled(t *testing.T) {
	err := mtls.Bind(mtls.PeerIdentity{Fingerprint: "aaa"}, mtls.BindRequest{
		ProjectID: "p", WorkerID: "w", HostID: "h", Fingerprint: "aaa", Status: "SUSPENDED",
	})
	if err == nil {
		t.Fatal()
	}
}

func TestExtractPeer_NoCert(t *testing.T) {
	_, err := mtls.ExtractPeerIdentity(tls.ConnectionState{})
	if err == nil {
		t.Fatal()
	}
}

func TestHandshake_mTLS_RoundTrip(t *testing.T) {
	ca, caKey, caPEM := genCA(t)
	srvPEM, srvKey, _ := genLeaf(t, ca, caKey, "controller.local", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	cliPEM, cliKey, cliCert := genLeaf(t, ca, caKey, "worker-1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))

	serverTLS, err := mtls.ServerTLSConfig(mtls.Config{CACertPEM: caPEM, CertPEM: srvPEM, KeyPEM: srvKey})
	if err != nil {
		t.Fatal(err)
	}
	clientTLS, err := mtls.ClientTLSConfig(mtls.Config{CACertPEM: caPEM, CertPEM: cliPEM, KeyPEM: cliKey, ServerName: "controller.local"})
	if err != nil {
		t.Fatal(err)
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		tc := c.(*tls.Conn)
		if err := tc.Handshake(); err != nil {
			done <- err
			return
		}
		peer, err := mtls.ExtractPeerIdentity(tc.ConnectionState())
		if err != nil {
			done <- err
			return
		}
		fp := mtls.FingerprintSHA256(cliCert)
		err = mtls.Bind(peer, mtls.BindRequest{
			ProjectID: "proj-a", WorkerID: "w1", HostID: "h1", Fingerprint: fp, Status: "ACTIVE",
		})
		done <- err
	}()

	conn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestHandshake_WrongCA_Fails(t *testing.T) {
	ca1, caKey1, caPEM1 := genCA(t)
	ca2, caKey2, _ := genCA(t)
	srvPEM, srvKey, _ := genLeaf(t, ca1, caKey1, "controller.local", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	cliPEM, cliKey, _ := genLeaf(t, ca2, caKey2, "worker-1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))

	serverTLS, _ := mtls.ServerTLSConfig(mtls.Config{CACertPEM: caPEM1, CertPEM: srvPEM, KeyPEM: srvKey})
	clientTLS, _ := mtls.ClientTLSConfig(mtls.Config{CACertPEM: caPEM1, CertPEM: cliPEM, KeyPEM: cliKey, ServerName: "controller.local"})

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	srvErr := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			srvErr <- err
			return
		}
		defer c.Close()
		srvErr <- c.(*tls.Conn).Handshake()
	}()
	conn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	if err != nil {
		return // client failed as expected
	}
	defer conn.Close()
	_ = conn.Handshake()
	if err := <-srvErr; err == nil {
		t.Fatal("server must reject client cert from wrong CA")
	}
}
