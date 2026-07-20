package acmeauthz

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

func TestInspectFetchesAuthorizationAndComputesDNS01Record(t *testing.T) {
	var baseURL string
	var receivedEnvelope acmejws.Envelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/directory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"newNonce":"` + baseURL + `/new-nonce",
				"newAccount":"` + baseURL + `/new-account",
				"newOrder":"` + baseURL + `/new-order",
				"meta":{"termsOfService":"` + baseURL + `/terms"}
			}`))
		case "/new-nonce":
			if r.Method != http.MethodHead {
				t.Fatalf("new nonce method = %s, want HEAD", r.Method)
			}
			w.Header().Set("Replay-Nonce", "authz-test-nonce")
			w.WriteHeader(http.StatusNoContent)
		case "/authz/1":
			if r.Method != http.MethodPost {
				t.Fatalf("authorization method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&receivedEnvelope); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"identifier":{"type":"dns","value":"example.com"},
				"status":"pending",
				"challenges":[
					{"type":"http-01","url":"` + baseURL + `/challenge/http","status":"pending","token":"http-token"},
					{"type":"dns-01","url":"` + baseURL + `/challenge/dns","status":"pending","token":"dns-token"}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL
	accountKey, err := acmeaccount.Open(filepath.Join(t.TempDir(), "account.key"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Inspect(context.Background(), Request{
		DirectoryURL:      server.URL + "/directory",
		AccountURL:        server.URL + "/account/1",
		AccountKey:        accountKey,
		AuthorizationURLs: []string{server.URL + "/authz/1"},
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Authorizations) != 1 {
		t.Fatalf("authorization count = %d, want 1", len(result.Authorizations))
	}
	authorization := result.Authorizations[0]
	if authorization.URL != server.URL+"/authz/1" || authorization.Domain != "example.com" || authorization.Status != "pending" {
		t.Fatalf("unexpected authorization: %#v", authorization)
	}
	if authorization.DNS01 == nil {
		t.Fatalf("dns-01 challenge missing: %#v", authorization)
	}
	if authorization.DNS01.RecordName != "_acme-challenge.example.com" || authorization.DNS01.RecordType != "TXT" || authorization.DNS01.RecordValue == "" {
		t.Fatalf("unexpected dns-01 record: %#v", authorization.DNS01)
	}
	protectedJSON, err := base64.RawURLEncoding.DecodeString(receivedEnvelope.Protected)
	if err != nil {
		t.Fatal(err)
	}
	protected := string(protectedJSON)
	if !strings.Contains(protected, `"kid":"`+server.URL+`/account/1"`) || strings.Contains(protected, `"jwk"`) {
		t.Fatalf("unexpected protected header: %s", protected)
	}
	if receivedEnvelope.Payload != "" {
		t.Fatalf("POST-as-GET payload = %q, want empty", receivedEnvelope.Payload)
	}
}

func TestInspectValidatesRequiredInputs(t *testing.T) {
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
				AccountURL:        "https://acme.example.test/account/1",
				AccountKey:        accountKey,
				AuthorizationURLs: []string{"https://acme.example.test/authz/1"},
			},
			want: "ACME directory URL 未配置",
		},
		{
			name: "missing account url",
			request: Request{
				DirectoryURL:      "https://acme.example.test/directory",
				AccountKey:        accountKey,
				AuthorizationURLs: []string{"https://acme.example.test/authz/1"},
			},
			want: "ACME account URL 未注册",
		},
		{
			name: "missing account key",
			request: Request{
				DirectoryURL:      "https://acme.example.test/directory",
				AccountURL:        "https://acme.example.test/account/1",
				AuthorizationURLs: []string{"https://acme.example.test/authz/1"},
			},
			want: "ACME account key 未加载",
		},
		{
			name: "missing authorization urls",
			request: Request{
				DirectoryURL: "https://acme.example.test/directory",
				AccountURL:   "https://acme.example.test/account/1",
				AccountKey:   accountKey,
			},
			want: "ACME order 尚未返回 authorization URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Inspect(context.Background(), tt.request, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Inspect error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
