package acmeregister

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
)

func TestRegisterCreatesACMEAccount(t *testing.T) {
	var directoryURL string
	var newNonceURL string
	var newAccountURL string
	var receivedEnvelope jwsEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/directory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"newNonce":"` + newNonceURL + `",
				"newAccount":"` + newAccountURL + `",
				"newOrder":"` + directoryURL + `/new-order",
				"meta":{"termsOfService":"` + directoryURL + `/terms"}
			}`))
		case "/new-nonce":
			if r.Method != http.MethodHead {
				t.Fatalf("new nonce method = %s, want HEAD", r.Method)
			}
			w.Header().Set("Replay-Nonce", "test-nonce")
			w.WriteHeader(http.StatusNoContent)
		case "/new-account":
			if r.Method != http.MethodPost {
				t.Fatalf("new account method = %s, want POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/jose+json" {
				t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
			}
			if err := json.NewDecoder(r.Body).Decode(&receivedEnvelope); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Location", directoryURL+"/account/1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"valid"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	directoryURL = server.URL
	newNonceURL = server.URL + "/new-nonce"
	newAccountURL = server.URL + "/new-account"
	accountKey, err := acmeaccount.Open(filepath.Join(t.TempDir(), "account.key"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Register(context.Background(), Request{
		DirectoryURL: server.URL + "/directory",
		ContactEmail: "Admin@Example.TEST",
		TermsAgreed:  true,
		AccountKey:   accountKey,
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountURL != server.URL+"/account/1" || result.ContactEmail != "admin@example.test" || result.Status != "valid" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if receivedEnvelope.Protected == "" || receivedEnvelope.Payload == "" || receivedEnvelope.Signature == "" {
		t.Fatalf("missing JWS fields: %#v", receivedEnvelope)
	}
	protectedJSON, err := base64.RawURLEncoding.DecodeString(receivedEnvelope.Protected)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(protectedJSON), `"alg":"ES256"`) || !strings.Contains(string(protectedJSON), `"nonce":"test-nonce"`) {
		t.Fatalf("unexpected protected header: %s", string(protectedJSON))
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(receivedEnvelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payloadJSON), `"mailto:admin@example.test"`) || !strings.Contains(string(payloadJSON), `"termsOfServiceAgreed":true`) {
		t.Fatalf("unexpected payload: %s", string(payloadJSON))
	}
}

func TestRegisterValidatesRequiredInputs(t *testing.T) {
	accountKey, err := acmeaccount.Open(filepath.Join(t.TempDir(), "account.key"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{
			name: "missing directory",
			request: Request{
				ContactEmail: "admin@example.test",
				TermsAgreed:  true,
				AccountKey:   accountKey,
			},
			want: "ACME directory URL 未配置",
		},
		{
			name: "missing contact email",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				TermsAgreed:  true,
				AccountKey:   accountKey,
			},
			want: "ACME 联系邮箱不能为空",
		},
		{
			name: "invalid contact email",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				ContactEmail: "invalid",
				TermsAgreed:  true,
				AccountKey:   accountKey,
			},
			want: "ACME 联系邮箱格式不正确",
		},
		{
			name: "terms not agreed",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				ContactEmail: "admin@example.test",
				AccountKey:   accountKey,
			},
			want: "需要先确认 ACME 服务条款",
		},
		{
			name: "missing account key",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				ContactEmail: "admin@example.test",
				TermsAgreed:  true,
			},
			want: "ACME account key 未加载",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Register(context.Background(), tt.request, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Register error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRegisterRejectsEABRequiredDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"newNonce":"` + "http://example.test/new-nonce" + `",
			"newAccount":"` + "http://example.test/new-account" + `",
			"newOrder":"` + "http://example.test/new-order" + `",
			"meta":{"externalAccountRequired":true}
		}`))
	}))
	defer server.Close()
	accountKey, err := acmeaccount.Open(filepath.Join(t.TempDir(), "account.key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Register(context.Background(), Request{
		DirectoryURL: server.URL,
		ContactEmail: "admin@example.test",
		TermsAgreed:  true,
		AccountKey:   accountKey,
	}, server.Client()); err == nil {
		t.Fatalf("Register should reject EAB-required directories")
	}
}
