package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestEncryptPlaintextDNSSecretsMigratesOnlyPlaintext(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.CreateDNSAccount(DNSAccount{
		ID:        "plain",
		Name:      "plain",
		Provider:  "alidns",
		SecretKey: "plain-secret",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateDNSAccount(DNSAccount{
		ID:        "encrypted",
		Name:      "encrypted",
		Provider:  "alidns",
		SecretKey: "enc:v1:already",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	count, err := st.EncryptPlaintextDNSSecrets(
		func(value string) (string, error) { return "enc:v1:" + value, nil },
		func(value string) bool { return len(value) >= 7 && value[:7] == "enc:v1:" },
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migrated count = %d, want 1", count)
	}
	plain, err := st.GetDNSAccount("plain")
	if err != nil {
		t.Fatal(err)
	}
	if plain.SecretKey != "enc:v1:plain-secret" {
		t.Fatalf("plain secret = %q", plain.SecretKey)
	}
	encrypted, err := st.GetDNSAccount("encrypted")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted.SecretKey != "enc:v1:already" {
		t.Fatalf("encrypted secret = %q", encrypted.SecretKey)
	}
}

func TestUpdateDNSAccountRejectsDuplicateName(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	first, err := st.CreateDNSAccount(DNSAccount{
		ID:        "first",
		Name:      "first",
		Provider:  "alidns",
		SecretKey: "enc:v1:first",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateDNSAccount(DNSAccount{
		ID:        "second",
		Name:      "second",
		Provider:  "alidns",
		SecretKey: "enc:v1:second",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	first.Name = "second"
	if _, err := st.UpdateDNSAccount(first); err != ErrAlreadyExists {
		t.Fatalf("duplicate update error = %v, want ErrAlreadyExists", err)
	}
}
