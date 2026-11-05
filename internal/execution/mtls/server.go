package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// OptionalClientCertTLSConfig builds the controller HTTPS configuration. Browser
// clients may omit a certificate; when a client certificate is supplied it is
// verified against the dedicated worker CA. Worker handlers still require a
// verified chain and bind the leaf fingerprint to the registered worker.
func OptionalClientCertTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS key pair: %w", err)
	}
	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
	}
	if caFile == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read worker CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("worker CA file contains no valid certificates")
	}
	cfg.ClientCAs = pool
	cfg.ClientAuth = tls.VerifyClientCertIfGiven
	return cfg, nil
}
