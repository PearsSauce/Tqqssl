package secretbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesKeyFileAndEncryptsRoundTrip(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "secret.key")
	box, err := Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions = %v, want 0600", info.Mode().Perm())
	}

	ciphertext, err := box.Encrypt("dns-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !IsCiphertext(ciphertext) || ciphertext == "dns-secret" {
		t.Fatalf("unexpected ciphertext: %q", ciphertext)
	}
	plaintext, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "dns-secret" {
		t.Fatalf("plaintext = %q, want dns-secret", plaintext)
	}

	reopened, err := Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err = reopened.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "dns-secret" {
		t.Fatalf("reopened plaintext = %q, want dns-secret", plaintext)
	}
}
