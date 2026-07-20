package certcrypto

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateKeyAndCSRIncludesAllDomains(t *testing.T) {
	bundle, err := GenerateKeyAndCSR([]string{"example.com", "www.example.com", "*.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.PrivateKeyPEM == "" || bundle.CSRPEM == "" {
		t.Fatalf("expected private key and csr pem, got %#v", bundle)
	}
	keyBlock, _ := pem.Decode([]byte(bundle.PrivateKeyPEM))
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" {
		t.Fatalf("unexpected key pem block: %#v", keyBlock)
	}
	if _, err := x509.ParseECPrivateKey(keyBlock.Bytes); err != nil {
		t.Fatalf("private key should parse as EC key: %v", err)
	}
	csrBlock, _ := pem.Decode([]byte(bundle.CSRPEM))
	if csrBlock == nil || csrBlock.Type != "CERTIFICATE REQUEST" {
		t.Fatalf("unexpected csr pem block: %#v", csrBlock)
	}
	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := csr.CheckSignature(); err != nil {
		t.Fatalf("csr signature invalid: %v", err)
	}
	want := []string{"example.com", "www.example.com", "*.example.com"}
	if len(csr.DNSNames) != len(want) {
		t.Fatalf("dns names = %#v, want %#v", csr.DNSNames, want)
	}
	for i := range want {
		if csr.DNSNames[i] != want[i] {
			t.Fatalf("dns names = %#v, want %#v", csr.DNSNames, want)
		}
	}
}

func TestGenerateKeyAndCSRRejectsEmptyDomains(t *testing.T) {
	if _, err := GenerateKeyAndCSR(nil); err == nil {
		t.Fatal("expected empty domain list to fail")
	}
}
