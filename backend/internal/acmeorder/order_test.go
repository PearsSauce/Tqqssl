package acmeorder

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
	"github.com/PearsSauce/Tqqssl/backend/internal/acmejws"
)

func TestCreateOrderPostsKIDJWSAndParsesResult(t *testing.T) {
	var directoryURL string
	var newNonceURL string
	var newOrderURL string
	var receivedEnvelope acmejws.Envelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/directory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"newNonce":"` + newNonceURL + `",
				"newAccount":"` + directoryURL + `/new-account",
				"newOrder":"` + newOrderURL + `",
				"meta":{"termsOfService":"` + directoryURL + `/terms"}
			}`))
		case "/new-nonce":
			if r.Method != http.MethodHead {
				t.Fatalf("new nonce method = %s, want HEAD", r.Method)
			}
			w.Header().Set("Replay-Nonce", "order-test-nonce")
			w.WriteHeader(http.StatusNoContent)
		case "/new-order":
			if r.Method != http.MethodPost {
				t.Fatalf("new order method = %s, want POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/jose+json" {
				t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
			}
			if err := json.NewDecoder(r.Body).Decode(&receivedEnvelope); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Location", directoryURL+"/order/1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"status":"pending",
				"authorizations":["` + directoryURL + `/authz/1","` + directoryURL + `/authz/2"],
				"finalize":"` + directoryURL + `/finalize/1",
				"expires":"2026-07-27T00:00:00Z"
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	directoryURL = server.URL
	newNonceURL = server.URL + "/new-nonce"
	newOrderURL = server.URL + "/new-order"
	accountKey, err := acmeaccount.Open(filepath.Join(t.TempDir(), "account.key"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Create(context.Background(), Request{
		DirectoryURL: server.URL + "/directory",
		AccountURL:   server.URL + "/account/1",
		AccountKey:   accountKey,
		Domains:      []string{"Example.COM", "www.example.com", "example.com"},
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if result.OrderURL != server.URL+"/order/1" || result.Status != "pending" || result.FinalizeURL != server.URL+"/finalize/1" {
		t.Fatalf("unexpected order result: %#v", result)
	}
	if len(result.AuthorizationURLs) != 2 || result.AuthorizationURLs[0] != server.URL+"/authz/1" || result.AuthorizationURLs[1] != server.URL+"/authz/2" {
		t.Fatalf("unexpected authorizations: %#v", result.AuthorizationURLs)
	}
	protectedJSON, err := base64.RawURLEncoding.DecodeString(receivedEnvelope.Protected)
	if err != nil {
		t.Fatal(err)
	}
	protected := string(protectedJSON)
	if !strings.Contains(protected, `"kid":"`+server.URL+`/account/1"`) || strings.Contains(protected, `"jwk"`) {
		t.Fatalf("unexpected protected header: %s", protected)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(receivedEnvelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(payloadJSON)
	if !strings.Contains(payload, `"value":"example.com"`) || !strings.Contains(payload, `"value":"www.example.com"`) {
		t.Fatalf("unexpected order payload: %s", payload)
	}
}

func TestCreateOrderValidatesRequiredInputs(t *testing.T) {
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
				AccountURL: "https://acme.example.test/account/1",
				AccountKey: accountKey,
				Domains:    []string{"example.com"},
			},
			want: "ACME directory URL 未配置",
		},
		{
			name: "missing account url",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				AccountKey:   accountKey,
				Domains:      []string{"example.com"},
			},
			want: "ACME account URL 未注册",
		},
		{
			name: "missing account key",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				AccountURL:   "https://acme.example.test/account/1",
				Domains:      []string{"example.com"},
			},
			want: "ACME account key 未加载",
		},
		{
			name: "missing domains",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				AccountURL:   "https://acme.example.test/account/1",
				AccountKey:   accountKey,
			},
			want: "ACME order 至少需要一个域名",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Create(context.Background(), tt.request, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Create error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
