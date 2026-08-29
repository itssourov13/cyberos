// Package mtls provides mutual TLS identity for remote workers (Phase 22A).
// mTLS authenticates the peer; it does NOT authorize execution.
// Authorization remains: Scope → Auth → CP → Grant → Broker → Host.
package mtls

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Config holds production mTLS settings. Private keys are never logged.
type Config struct {
	// CACertPEM is the trust anchor for peer certificates.
	CACertPEM []byte
	// CertPEM is this side's certificate (server or client).
	CertPEM []byte
	// KeyPEM is this side's private key (never persisted to DB/events).
	KeyPEM []byte
	// ServerName is expected for client-side verification of the controller.
	ServerName string
	// MinVersion defaults to TLS 1.3.
	MinVersion uint16
}

// Validate rejects insecure configurations.
func (c Config) Validate() error {
	if len(c.CACertPEM) == 0 {
		return fmt.Errorf("mtls: CA certificate required")
	}
	if len(c.CertPEM) == 0 || len(c.KeyPEM) == 0 {
		return fmt.Errorf("mtls: certificate and private key required")
	}
	return nil
}

// ServerTLSConfig builds a strict mTLS server config (RequireAndVerifyClientCert).
func ServerTLSConfig(c Config) (*tls.Config, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(c.CertPEM, c.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("mtls: server key pair: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(c.CACertPEM) {
		return nil, fmt.Errorf("mtls: invalid CA PEM")
	}
	min := c.MinVersion
	if min == 0 {
		min = tls.VersionTLS13
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   min,
		// Never skip verification.
	}, nil
}

// ClientTLSConfig builds a strict mTLS client config for workers.
func ClientTLSConfig(c Config) (*tls.Config, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(c.CertPEM, c.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("mtls: client key pair: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(c.CACertPEM) {
		return nil, fmt.Errorf("mtls: invalid CA PEM")
	}
	min := c.MinVersion
	if min == 0 {
		min = tls.VersionTLS13
	}
	if c.ServerName == "" {
		return nil, fmt.Errorf("mtls: ServerName required for client verification")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   c.ServerName,
		MinVersion:   min,
		// InsecureSkipVerify must remain false.
	}, nil
}

// FingerprintSHA256 returns lowercase hex SHA-256 of the DER certificate.
func FingerprintSHA256(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

// PeerIdentity is derived solely from the verified TLS client certificate.
type PeerIdentity struct {
	Fingerprint string
	Subject     string
	NotBefore   time.Time
	NotAfter    time.Time
}

// ExtractPeerIdentity requires a completed handshake with peer certificates.
func ExtractPeerIdentity(state tls.ConnectionState) (PeerIdentity, error) {
	if len(state.PeerCertificates) == 0 {
		return PeerIdentity{}, fmt.Errorf("mtls: WORKER_CERT_REQUIRED")
	}
	leaf := state.PeerCertificates[0]
	now := time.Now().UTC()
	if now.Before(leaf.NotBefore) {
		return PeerIdentity{}, fmt.Errorf("mtls: WORKER_CERT_NOT_YET_VALID")
	}
	if now.After(leaf.NotAfter) {
		return PeerIdentity{}, fmt.Errorf("mtls: WORKER_CERT_EXPIRED")
	}
	return PeerIdentity{
		Fingerprint: FingerprintSHA256(leaf),
		Subject:     leaf.Subject.String(),
		NotBefore:   leaf.NotBefore.UTC(),
		NotAfter:    leaf.NotAfter.UTC(),
	}, nil
}

// BindRequest is the expected registered worker identity (server-side authority).
type BindRequest struct {
	ProjectID   string
	WorkerID    string
	HostID      string
	Fingerprint string // registered cert fingerprint (required for bind)
	Status      string // ACTIVE required
}

// Bind verifies TLS peer fingerprint matches registered worker identity.
// Does NOT authorize execution.
func Bind(peer PeerIdentity, req BindRequest) error {
	if peer.Fingerprint == "" {
		return fmt.Errorf("mtls: WORKER_CERT_INVALID")
	}
	if req.Fingerprint == "" {
		return fmt.Errorf("mtls: WORKER_IDENTITY_UNKNOWN")
	}
	if !strings.EqualFold(peer.Fingerprint, req.Fingerprint) {
		return fmt.Errorf("mtls: WORKER_IDENTITY_MISMATCH")
	}
	if req.WorkerID == "" || req.ProjectID == "" || req.HostID == "" {
		return fmt.Errorf("mtls: WORKER_IDENTITY_UNKNOWN")
	}
	switch strings.ToUpper(req.Status) {
	case "ACTIVE":
		// ok
	case "REVOKED":
		return fmt.Errorf("mtls: WORKER_REVOKED")
	case "SUSPENDED", "DISABLED":
		return fmt.Errorf("mtls: WORKER_DISABLED")
	case "EXPIRED":
		return fmt.Errorf("mtls: WORKER_EXPIRED")
	default:
		return fmt.Errorf("mtls: WORKER_DISABLED")
	}
	return nil
}
