package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/config"
	"github.com/PearsSauce/Tqqssl/backend/internal/store"
)

func TestAuthRegisterLoginMeAndLogout(t *testing.T) {
	handler := newTestHandler(t)

	optionsRec := request(handler, http.MethodGet, "/api/v1/auth/register/options", "", nil)
	if optionsRec.Code != http.StatusOK || !strings.Contains(optionsRec.Body.String(), `"allowRegister":true`) {
		t.Fatalf("register options = %d %s", optionsRec.Code, optionsRec.Body.String())
	}

	registerRec := request(handler, http.MethodPost, "/api/v1/auth/register", `{
		"username":"admin",
		"email":"admin@example.test",
		"password":"AdminPassw0rd!"
	}`, map[string]string{"X-Forwarded-Proto": "https"})
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", registerRec.Code, registerRec.Body.String())
	}
	registerCookie := findCookie(registerRec.Result().Cookies(), sessionCookieName)
	if registerCookie == nil || registerCookie.Value == "" {
		t.Fatalf("register did not set session cookie: %#v", registerRec.Result().Cookies())
	}
	if !registerCookie.HttpOnly || !registerCookie.Secure || registerCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security attrs wrong: %#v", registerCookie)
	}
	var registered struct {
		User UserDTO `json:"user"`
	}
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered.User.Role != "admin" || registered.User.ID == "" || !strings.HasPrefix(registered.User.ID[14:], "7") {
		t.Fatalf("registered user should be uuidv7 admin: %#v", registered.User)
	}

	secondRec := request(handler, http.MethodPost, "/api/v1/auth/register", `{
		"username":"second",
		"email":"second@example.test",
		"password":"SecondPassw0rd!"
	}`, nil)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("second register = %d %s, want 409", secondRec.Code, secondRec.Body.String())
	}

	meRec := requestWithCookie(handler, http.MethodGet, "/api/v1/auth/me", "", registerCookie)
	if meRec.Code != http.StatusOK || !strings.Contains(meRec.Body.String(), `"username":"admin"`) {
		t.Fatalf("me = %d %s", meRec.Code, meRec.Body.String())
	}

	logoutRec := requestWithCookie(handler, http.MethodPost, "/api/v1/auth/logout", "", registerCookie)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", logoutRec.Code, logoutRec.Body.String())
	}
	clearedCookie := findCookie(logoutRec.Result().Cookies(), sessionCookieName)
	if clearedCookie == nil || clearedCookie.MaxAge >= 0 {
		t.Fatalf("logout should clear cookie: %#v", logoutRec.Result().Cookies())
	}

	loginRec := request(handler, http.MethodPost, "/api/v1/auth/login", `{
		"username":"admin@example.test",
		"password":"AdminPassw0rd!"
	}`, map[string]string{"X-Forwarded-Proto": "https"})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login = %d %s", loginRec.Code, loginRec.Body.String())
	}
	if findCookie(loginRec.Result().Cookies(), sessionCookieName) == nil {
		t.Fatalf("login should set session cookie")
	}
}

func TestAuthRejectsWeakPasswordAndBadLogin(t *testing.T) {
	handler := newTestHandler(t)

	weakRec := request(handler, http.MethodPost, "/api/v1/auth/register", `{
		"username":"admin",
		"email":"admin@example.test",
		"password":"short"
	}`, nil)
	if weakRec.Code != http.StatusBadRequest {
		t.Fatalf("weak register = %d %s, want 400", weakRec.Code, weakRec.Body.String())
	}

	loginRec := request(handler, http.MethodPost, "/api/v1/auth/login", `{
		"username":"missing@example.test",
		"password":"AdminPassw0rd!"
	}`, nil)
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login = %d %s, want 401", loginRec.Code, loginRec.Body.String())
	}
}

func TestProtectedDNSAndCertificateApplicationFlow(t *testing.T) {
	handler := newTestHandler(t)

	unauthorizedRec := request(handler, http.MethodGet, "/api/v1/dns-accounts", "", nil)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized dns list = %d %s, want 401", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	cookie := registerAdmin(t, handler)
	createDNSRec := requestWithCookie(handler, http.MethodPost, "/api/v1/dns-accounts", `{
		"name":"AliDNS 主账号",
		"provider":"AliDNS",
		"accessKey":"AKIDEXAMPLE1234567890",
		"secretKey":"VerySecretValueShouldNotLeak",
		"remark":"个人版本地测试账号"
	}`, cookie)
	if createDNSRec.Code != http.StatusCreated {
		t.Fatalf("create dns account = %d %s", createDNSRec.Code, createDNSRec.Body.String())
	}
	if strings.Contains(createDNSRec.Body.String(), "VerySecretValueShouldNotLeak") || strings.Contains(createDNSRec.Body.String(), "secretKey") {
		t.Fatalf("dns dto leaked secret: %s", createDNSRec.Body.String())
	}
	var dnsAccount DNSAccountDTO
	if err := json.Unmarshal(createDNSRec.Body.Bytes(), &dnsAccount); err != nil {
		t.Fatal(err)
	}
	if dnsAccount.ID == "" || dnsAccount.Provider != "alidns" || !dnsAccount.HasSecretKey || dnsAccount.AccessKeyMasked == "" {
		t.Fatalf("unexpected dns dto: %#v", dnsAccount)
	}

	listDNSRec := requestWithCookie(handler, http.MethodGet, "/api/v1/dns-accounts", "", cookie)
	if listDNSRec.Code != http.StatusOK {
		t.Fatalf("list dns accounts = %d %s", listDNSRec.Code, listDNSRec.Body.String())
	}
	if strings.Contains(listDNSRec.Body.String(), "VerySecretValueShouldNotLeak") || strings.Contains(listDNSRec.Body.String(), "secretKey") {
		t.Fatalf("dns list leaked secret: %s", listDNSRec.Body.String())
	}

	missingDNSRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications", `{
		"primaryDomain":"example.com",
		"sans":[],
		"dnsAccountId":"missing",
		"challengeMode":"dns-01"
	}`, cookie)
	if missingDNSRec.Code != http.StatusBadRequest {
		t.Fatalf("missing dns certificate application = %d %s, want 400", missingDNSRec.Code, missingDNSRec.Body.String())
	}

	createCertificateRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications", `{
		"primaryDomain":"Example.COM.",
		"sans":["www.example.com","*.example.com","www.example.com"],
		"dnsAccountId":"`+dnsAccount.ID+`",
		"challengeMode":"dns-01"
	}`, cookie)
	if createCertificateRec.Code != http.StatusCreated {
		t.Fatalf("create certificate application = %d %s", createCertificateRec.Code, createCertificateRec.Body.String())
	}
	var application CertificateApplicationDTO
	if err := json.Unmarshal(createCertificateRec.Body.Bytes(), &application); err != nil {
		t.Fatal(err)
	}
	if application.ID == "" || application.PrimaryDomain != "example.com" || application.ChallengeMode != challengeModeDNS01 || application.Status != certificatePending {
		t.Fatalf("unexpected certificate application dto: %#v", application)
	}
	if len(application.SANs) != 2 || application.SANs[0] != "www.example.com" || application.SANs[1] != "*.example.com" {
		t.Fatalf("unexpected normalized sans: %#v", application.SANs)
	}

	deleteDNSRec := requestWithCookie(handler, http.MethodDelete, "/api/v1/dns-accounts/"+dnsAccount.ID, "", cookie)
	if deleteDNSRec.Code != http.StatusConflict {
		t.Fatalf("delete dns in use = %d %s, want 409", deleteDNSRec.Code, deleteDNSRec.Body.String())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	api := New(config.Config{FrontendOrigin: "https://app.example.test", SessionTTL: time.Hour}, st, nil)
	return api.Routes()
}

func registerAdmin(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()
	rec := request(handler, http.MethodPost, "/api/v1/auth/register", `{
		"username":"admin",
		"email":"admin@example.test",
		"password":"AdminPassw0rd!"
	}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register admin = %d %s", rec.Code, rec.Body.String())
	}
	cookie := findCookie(rec.Result().Cookies(), sessionCookieName)
	if cookie == nil || cookie.Value == "" {
		t.Fatalf("register admin did not set session cookie: %#v", rec.Result().Cookies())
	}
	return cookie
}

func request(handler http.Handler, method string, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func requestWithCookie(handler http.Handler, method string, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
