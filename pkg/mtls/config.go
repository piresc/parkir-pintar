// Package mtls provides mutual TLS configuration for inter-service communication.
package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config holds mTLS configuration parameters.
type Config struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	CAFile   string
}

// FromEnv loads mTLS configuration from environment variables.
// Environment variables:
//   - MTLS_ENABLED: "true" to enable mTLS (default: false)
//   - MTLS_CERT_FILE: path to the TLS certificate
//   - MTLS_KEY_FILE: path to the TLS private key
//   - MTLS_CA_FILE: path to the CA certificate
func FromEnv() Config {
	return Config{
		Enabled:  os.Getenv("MTLS_ENABLED") == "true",
		CertFile: os.Getenv("MTLS_CERT_FILE"),
		KeyFile:  os.Getenv("MTLS_KEY_FILE"),
		CAFile:   os.Getenv("MTLS_CA_FILE"),
	}
}

// LoadTLSConfig loads a base TLS configuration with the given cert, key, and CA.
// Returns nil if mTLS is disabled via MTLS_ENABLED env var.
func LoadTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cfg := FromEnv()
	if !cfg.Enabled {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to load key pair: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("mtls: failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadServerTLSConfig returns a TLS config suitable for gRPC servers.
// It requires client certificates (mutual TLS).
// Returns nil, nil if MTLS_ENABLED is not "true".
func LoadServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if os.Getenv("MTLS_ENABLED") != "true" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to load server key pair: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("mtls: failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadClientTLSConfig returns a TLS config suitable for gRPC clients.
// It presents a client certificate and verifies the server against the CA.
// Returns nil, nil if MTLS_ENABLED is not "true".
func LoadClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if os.Getenv("MTLS_ENABLED") != "true" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to load client key pair: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("mtls: failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("mtls: failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
