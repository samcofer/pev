package primitives

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/discover"
)

// makeCertKey builds a self-signed RSA cert and returns paths to the PEM cert
// and PEM private key. Used to drive x509 primitive tests without bringing
// in real certificates.
func makeCertKey(t *testing.T, hostname string, notAfter time.Time) (certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		DNSNames:     []string{hostname},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func TestX509CertKeyPairing(t *testing.T) {
	certA, keyA := makeCertKey(t, "host.example.com", time.Now().Add(365*24*time.Hour))
	_, keyB := makeCertKey(t, "host.example.com", time.Now().Add(365*24*time.Hour))

	pass := checks.Check{
		ID: "x", Title: "x", Primitive: "x509",
		With: map[string]interface{}{"cert_path": certA, "key_path": keyA, "match_hostname": "host.example.com", "not_after_min_days": 30},
	}
	if r := runRC(t, pass, discover.HostFacts{}); r.Status != checks.StatusPass {
		t.Fatalf("expected pass, got %s/%s", r.Status, r.Reason)
	}

	mismatch := checks.Check{
		ID: "x", Title: "x", Primitive: "x509",
		With: map[string]interface{}{"cert_path": certA, "key_path": keyB},
	}
	r := runRC(t, mismatch, discover.HostFacts{})
	if r.Status != checks.StatusFail || !strings.Contains(r.Reason, "mismatch") {
		t.Fatalf("expected modulus mismatch, got %s/%s", r.Status, r.Reason)
	}
}

func TestX509HostnameMismatch(t *testing.T) {
	cert, key := makeCertKey(t, "host.example.com", time.Now().Add(365*24*time.Hour))
	c := checks.Check{
		ID: "x", Title: "x", Primitive: "x509",
		With: map[string]interface{}{"cert_path": cert, "key_path": key, "match_hostname": "other.example.com"},
	}
	r := runRC(t, c, discover.HostFacts{})
	if r.Status != checks.StatusFail || !strings.Contains(r.Reason, "hostname") {
		t.Fatalf("expected hostname mismatch, got %s/%s", r.Status, r.Reason)
	}
}

func TestX509ExpirationCushion(t *testing.T) {
	// Cert expires in 5 days; the check requires at least 30.
	cert, key := makeCertKey(t, "host.example.com", time.Now().Add(5*24*time.Hour))
	c := checks.Check{
		ID: "x", Title: "x", Primitive: "x509",
		With: map[string]interface{}{"cert_path": cert, "key_path": key, "not_after_min_days": 30},
	}
	r := runRC(t, c, discover.HostFacts{})
	if r.Status != checks.StatusFail || !strings.Contains(r.Reason, "expires") {
		t.Fatalf("expected expiration failure, got %s/%s", r.Status, r.Reason)
	}
}
