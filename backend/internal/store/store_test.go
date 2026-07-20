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

func TestDeleteCertificateApplicationUnblocksDNSAccountDeletion(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.CreateDNSAccount(DNSAccount{
		ID:        "dns",
		Name:      "dns",
		Provider:  "alidns",
		SecretKey: "enc:v1:dns",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateCertificateApplication(CertificateApplication{
		ID:            "cert",
		PrimaryDomain: "example.com",
		DNSAccountID:  "dns",
		ChallengeMode: "dns-01",
		Status:        "pending",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteDNSAccount("dns"); err != ErrInUse {
		t.Fatalf("delete dns in use error = %v, want ErrInUse", err)
	}
	if err := st.DeleteCertificateApplication("cert"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteDNSAccount("dns"); err != nil {
		t.Fatalf("delete dns after certificate deletion = %v", err)
	}
}

func TestSaveCertificateApplicationOrder(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.CreateDNSAccount(DNSAccount{
		ID:        "dns",
		Name:      "dns",
		Provider:  "alidns",
		SecretKey: "enc:v1:dns",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateCertificateApplication(CertificateApplication{
		ID:            "cert",
		PrimaryDomain: "example.com",
		SANs:          []string{"www.example.com"},
		DNSAccountID:  "dns",
		ChallengeMode: "dns-01",
		Status:        "pending",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	updatedAt := now.Add(time.Minute)
	application, err := st.SaveCertificateApplicationOrder("cert", "ordered", CertificateOrder{
		OrderURL:          "https://acme.example.test/order/1",
		OrderStatus:       "pending",
		AuthorizationURLs: []string{"https://acme.example.test/authz/1"},
		FinalizeURL:       "https://acme.example.test/finalize/1",
	}, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if application.Status != "ordered" || application.OrderURL == "" || application.OrderStatus != "pending" || application.FinalizeURL == "" {
		t.Fatalf("unexpected ordered application: %#v", application)
	}
	if len(application.AuthorizationURLs) != 1 || application.AuthorizationURLs[0] != "https://acme.example.test/authz/1" {
		t.Fatalf("unexpected authorization urls: %#v", application.AuthorizationURLs)
	}
	application.AuthorizationURLs[0] = "mutated"
	loaded, err := st.GetCertificateApplication("cert")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AuthorizationURLs[0] != "https://acme.example.test/authz/1" {
		t.Fatalf("authorization urls should be copied: %#v", loaded.AuthorizationURLs)
	}
}

func TestSaveAndGetACMEAccount(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetACMEAccount(); err != ErrNotFound {
		t.Fatalf("empty acme account error = %v, want ErrNotFound", err)
	}
	now := time.Now().UTC()
	account, err := st.SaveACMEAccount(ACMEAccount{
		DirectoryURL: "https://acme.example.test/directory",
		AccountURL:   "https://acme.example.test/account/1",
		ContactEmail: "admin@example.test",
		Status:       "valid",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := st.GetACMEAccount()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccountURL != account.AccountURL || loaded.ContactEmail != "admin@example.test" || loaded.Status != "valid" {
		t.Fatalf("unexpected loaded acme account: %#v", loaded)
	}
}
