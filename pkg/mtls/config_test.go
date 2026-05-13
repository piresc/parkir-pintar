package mtls

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
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateTestCerts creates a self-signed CA and a leaf certificate for testing.
// Returns paths to ca.pem, cert.pem, key.pem.
func generateTestCerts(t *testing.T) (caFile, certFile, keyFile string) {
	t.Helper()
	dir := t.TempDir()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	// CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"ParkIR Pintar Test CA"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("failed to parse CA cert: %v", err)
	}

	// Write CA cert
	caFile = filepath.Join(dir, "ca.pem")
	writePEM(t, caFile, "CERTIFICATE", caCertDER)

	// Generate leaf key
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate leaf key: %v", err)
	}

	// Leaf certificate template
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"ParkIR Pintar Test Service"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
	}

	leafCertDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create leaf cert: %v", err)
	}

	// Write leaf cert
	certFile = filepath.Join(dir, "cert.pem")
	writePEM(t, certFile, "CERTIFICATE", leafCertDER)

	// Write leaf key
	keyFile = filepath.Join(dir, "key.pem")
	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatalf("failed to marshal leaf key: %v", err)
	}
	writePEM(t, keyFile, "EC PRIVATE KEY", leafKeyDER)

	return caFile, certFile, keyFile
}

func writePEM(t *testing.T, path, blockType string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
	defer f.Close()

	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		t.Fatalf("failed to write PEM to %s: %v", path, err)
	}
}

func TestLoadServerTLSConfig_Disabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "false")

	cfg, err := LoadServerTLSConfig("cert.pem", "key.pem", "ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when mTLS is disabled")
	}
}

func TestLoadClientTLSConfig_Disabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "false")

	cfg, err := LoadClientTLSConfig("cert.pem", "key.pem", "ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when mTLS is disabled")
	}
}

func TestLoadTLSConfig_Disabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "false")

	cfg, err := LoadTLSConfig("cert.pem", "key.pem", "ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when mTLS is disabled")
	}
}

func TestLoadServerTLSConfig_Enabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	caFile, certFile, keyFile := generateTestCerts(t)

	cfg, err := LoadServerTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("expected RequireAndVerifyClientCert, got %v", cfg.ClientAuth)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 minimum, got %d", cfg.MinVersion)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set")
	}
}

func TestLoadClientTLSConfig_Enabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	caFile, certFile, keyFile := generateTestCerts(t)

	cfg, err := LoadClientTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 minimum, got %d", cfg.MinVersion)
	}
}

func TestLoadTLSConfig_Enabled(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	caFile, certFile, keyFile := generateTestCerts(t)

	cfg, err := LoadTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
	if cfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
}

func TestLoadServerTLSConfig_InvalidCert(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")

	_, err := LoadServerTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem", "/nonexistent/ca.pem")
	if err == nil {
		t.Fatal("expected error for invalid cert paths")
	}
}

func TestLoadClientTLSConfig_InvalidCert(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")

	_, err := LoadClientTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem", "/nonexistent/ca.pem")
	if err == nil {
		t.Fatal("expected error for invalid cert paths")
	}
}

func TestLoadServerTLSConfig_InvalidCA(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	_, certFile, keyFile := generateTestCerts(t)

	// Write invalid CA file
	invalidCA := filepath.Join(t.TempDir(), "bad-ca.pem")
	if err := os.WriteFile(invalidCA, []byte("not a cert"), 0o600); err != nil {
		t.Fatalf("failed to write invalid CA: %v", err)
	}

	_, err := LoadServerTLSConfig(certFile, keyFile, invalidCA)
	if err == nil {
		t.Fatal("expected error for invalid CA")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	t.Setenv("MTLS_CERT_FILE", "/certs/cert.pem")
	t.Setenv("MTLS_KEY_FILE", "/certs/key.pem")
	t.Setenv("MTLS_CA_FILE", "/certs/ca.pem")

	cfg := FromEnv()
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.CertFile != "/certs/cert.pem" {
		t.Errorf("expected CertFile=/certs/cert.pem, got %s", cfg.CertFile)
	}
	if cfg.KeyFile != "/certs/key.pem" {
		t.Errorf("expected KeyFile=/certs/key.pem, got %s", cfg.KeyFile)
	}
	if cfg.CAFile != "/certs/ca.pem" {
		t.Errorf("expected CAFile=/certs/ca.pem, got %s", cfg.CAFile)
	}
}

func TestMTLS_EndToEnd(t *testing.T) {
	t.Setenv("MTLS_ENABLED", "true")
	caFile, certFile, keyFile := generateTestCerts(t)

	serverCfg, err := LoadServerTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("server config error: %v", err)
	}

	clientCfg, err := LoadClientTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("client config error: %v", err)
	}

	// Start TLS server
	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 5)
		n, _ := conn.Read(buf)
		conn.Write(buf[:n])
	}()

	// Connect with client
	clientCfg.ServerName = "localhost"
	conn, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
	if err != nil {
		t.Fatalf("client dial error: %v", err)
	}
	defer conn.Close()

	msg := []byte("hello")
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write error: %v", err)
	}

	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("expected 'hello', got %q", string(buf[:n]))
	}

	<-done
}
