package config

import "testing"

func TestLoadACMEDefaultsAreSafe(t *testing.T) {
	t.Setenv("TQQSSL_ACME_DIRECTORY_URL", "")
	t.Setenv("TQQSSL_ACME_TERMS_AGREED", "")

	cfg := Load()
	if cfg.ACMEDirectoryURL != "" {
		t.Fatalf("ACMEDirectoryURL = %q, want empty", cfg.ACMEDirectoryURL)
	}
	if cfg.ACMETermsAgreed {
		t.Fatalf("ACMETermsAgreed should default to false")
	}
}

func TestLoadACMEEnvOverrides(t *testing.T) {
	t.Setenv("TQQSSL_ACME_DIRECTORY_URL", "https://acme.example.test/directory")
	t.Setenv("TQQSSL_ACME_TERMS_AGREED", "true")

	cfg := Load()
	if cfg.ACMEDirectoryURL != "https://acme.example.test/directory" {
		t.Fatalf("ACMEDirectoryURL = %q", cfg.ACMEDirectoryURL)
	}
	if !cfg.ACMETermsAgreed {
		t.Fatalf("ACMETermsAgreed should be true")
	}
}
