package acmedirectory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckDirectorySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"newNonce": "https://acme.example.test/new-nonce",
			"newAccount": "https://acme.example.test/new-account",
			"newOrder": "https://acme.example.test/new-order",
			"extraEndpoint": "https://acme.example.test/extra",
			"meta": {
				"termsOfService": "https://acme.example.test/terms",
				"website": "https://acme.example.test",
				"externalAccountRequired": true
			}
		}`))
	}))
	defer server.Close()

	result, err := Check(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if result.DirectoryURL != server.URL || result.NewNonce == "" || result.NewAccount == "" || result.NewOrder == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.TermsOfService != "https://acme.example.test/terms" || result.Website != "https://acme.example.test" {
		t.Fatalf("unexpected meta: %#v", result)
	}
	if !result.ExternalAccountRequired || len(result.Warnings) == 0 {
		t.Fatalf("expected external account warning: %#v", result)
	}
}

func TestCheckDirectoryRequiresCoreEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"newNonce":"https://acme.example.test/new-nonce"}`))
	}))
	defer server.Close()

	if _, err := Check(context.Background(), server.URL, server.Client()); err == nil {
		t.Fatalf("Check should reject directory without core endpoints")
	}
}

func TestCheckDirectoryRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	if _, err := Check(context.Background(), server.URL, server.Client()); err == nil {
		t.Fatalf("Check should reject non-2xx directory response")
	}
}
