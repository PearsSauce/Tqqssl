package acmeaccount

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCreatesAndReloadsP256AccountKey(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "acme-account.key")
	accountKey, err := Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if accountKey.Type() != keyType {
		t.Fatalf("key type = %q, want %q", accountKey.Type(), keyType)
	}
	if accountKey.Path() != keyFile {
		t.Fatalf("key path = %q, want %q", accountKey.Path(), keyFile)
	}
	publicKey := accountKey.PublicKey()
	if publicKey.X == nil || publicKey.Y == nil {
		t.Fatalf("public key should be populated")
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions = %v, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "BEGIN EC PRIVATE KEY") {
		t.Fatalf("key file should be EC private key PEM")
	}

	reloaded, err := Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	reloadedPublicKey := reloaded.PublicKey()
	if publicKey.X.Cmp(reloadedPublicKey.X) != 0 || publicKey.Y.Cmp(reloadedPublicKey.Y) != 0 {
		t.Fatalf("reloaded key does not match created key")
	}
}

func TestOpenRejectsInvalidKeyFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "acme-account.key")
	if err := os.WriteFile(keyFile, []byte("not a key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(keyFile); err == nil {
		t.Fatalf("Open should reject invalid key file")
	}
}
