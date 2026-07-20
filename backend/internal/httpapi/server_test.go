package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
	"github.com/PearsSauce/Tqqssl/backend/internal/config"
	"github.com/PearsSauce/Tqqssl/backend/internal/secretbox"
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

func TestACMEStatusRequiresAuthAndReportsReadiness(t *testing.T) {
	handler := newTestHandler(t)

	unauthorizedRec := request(handler, http.MethodGet, "/api/v1/acme/status", "", nil)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized acme status = %d %s, want 401", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	cookie := registerAdmin(t, handler)
	statusRec := requestWithCookie(handler, http.MethodGet, "/api/v1/acme/status", "", cookie)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("acme status = %d %s", statusRec.Code, statusRec.Body.String())
	}
	var status ACMEStatusDTO
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.AccountKeyReady || status.AccountKeyType != "ECDSA P-256" || !status.TermsAgreed || !status.Ready {
		t.Fatalf("unexpected acme status: %#v", status)
	}
	if status.DirectoryURL != "https://acme.example.test/directory" {
		t.Fatalf("directory url = %q", status.DirectoryURL)
	}
	if strings.Contains(statusRec.Body.String(), "key_file") || strings.Contains(statusRec.Body.String(), "PRIVATE KEY") {
		t.Fatalf("acme status leaked sensitive key material: %s", statusRec.Body.String())
	}
}

func TestACMEDirectoryCheckRequiresAuthAndValidatesDirectory(t *testing.T) {
	directoryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"newNonce":"` + "http://example.test/new-nonce" + `",
			"newAccount":"` + "http://example.test/new-account" + `",
			"newOrder":"` + "http://example.test/new-order" + `",
			"meta":{"termsOfService":"` + "http://example.test/terms" + `"}
		}`))
	}))
	defer directoryServer.Close()
	handler, _ := newTestHandlerWithDirectory(t, directoryServer.URL)

	unauthorizedRec := request(handler, http.MethodPost, "/api/v1/acme/directory/check", "", nil)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized acme directory check = %d %s, want 401", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	cookie := registerAdmin(t, handler)
	checkRec := requestWithCookie(handler, http.MethodPost, "/api/v1/acme/directory/check", "", cookie)
	if checkRec.Code != http.StatusOK {
		t.Fatalf("acme directory check = %d %s", checkRec.Code, checkRec.Body.String())
	}
	var result ACMEDirectoryCheckDTO
	if err := json.Unmarshal(checkRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.DirectoryURL != directoryServer.URL || result.NewNonce == "" || result.NewAccount == "" || result.NewOrder == "" {
		t.Fatalf("unexpected directory check result: %#v", result)
	}
	if result.TermsOfService != "http://example.test/terms" {
		t.Fatalf("terms of service = %q", result.TermsOfService)
	}
}

func TestRegisterACMEAccountPersistsStatus(t *testing.T) {
	var baseURL string
	acmeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Replay-Nonce", "register-test-nonce")
			w.WriteHeader(http.StatusNoContent)
		case "/new-account":
			if r.Method != http.MethodPost {
				t.Fatalf("new account method = %s, want POST", r.Method)
			}
			var envelope struct {
				Protected string `json:"protected"`
				Payload   string `json:"payload"`
				Signature string `json:"signature"`
			}
			if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Protected == "" || envelope.Payload == "" || envelope.Signature == "" {
				t.Fatalf("missing jws fields: %#v", envelope)
			}
			w.Header().Set("Location", baseURL+"/account/1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"valid"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer acmeServer.Close()
	baseURL = acmeServer.URL
	handler, _ := newTestHandlerWithDirectory(t, acmeServer.URL+"/directory")
	cookie := registerAdmin(t, handler)

	registerRec := requestWithCookie(handler, http.MethodPost, "/api/v1/acme/account/register", "{}", cookie)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register acme account = %d %s", registerRec.Code, registerRec.Body.String())
	}
	if strings.Contains(registerRec.Body.String(), "PRIVATE KEY") {
		t.Fatalf("register response leaked private key: %s", registerRec.Body.String())
	}
	var registered struct {
		AccountRegistered bool   `json:"accountRegistered"`
		AccountURL        string `json:"accountUrl"`
		AccountStatus     string `json:"accountStatus"`
		ContactEmail      string `json:"contactEmail"`
	}
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if !registered.AccountRegistered || registered.AccountURL != acmeServer.URL+"/account/1" || registered.ContactEmail != "admin@example.test" || registered.AccountStatus != "valid" {
		t.Fatalf("unexpected register response: %#v", registered)
	}

	statusRec := requestWithCookie(handler, http.MethodGet, "/api/v1/acme/status", "", cookie)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("acme status after registration = %d %s", statusRec.Code, statusRec.Body.String())
	}
	var status ACMEStatusDTO
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.AccountRegistered || status.AccountURL != registered.AccountURL || status.ContactEmail != "admin@example.test" || status.AccountStatus != "valid" {
		t.Fatalf("unexpected acme status after registration: %#v", status)
	}
}

func TestRegisterACMEAccountReturnsPersistedStatus(t *testing.T) {
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "store.json")
	keyFile := filepath.Join(dir, "secret.key")
	st, err := store.Open(dataFile)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.SaveACMEAccount(store.ACMEAccount{
		DirectoryURL: "https://acme.example.test/directory",
		AccountURL:   "https://acme.example.test/account/1",
		ContactEmail: "admin@example.test",
		Status:       "valid",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	box, err := secretbox.Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	api := New(config.Config{
		FrontendOrigin:   "https://app.example.test",
		SecretKeyFile:    keyFile,
		ACMEDirectoryURL: "http://127.0.0.1:1/directory",
		SessionTTL:       time.Hour,
	}, st, box, nil, nil)
	handler := api.Routes()
	cookie := registerAdmin(t, handler)

	registerRec := requestWithCookie(handler, http.MethodPost, "/api/v1/acme/account/register", `{"contactEmail":"other@example.test"}`, cookie)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register existing acme account = %d %s", registerRec.Code, registerRec.Body.String())
	}
	var registered ACMEAccountRegistrationDTO
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if !registered.AccountRegistered || registered.AccountURL != "https://acme.example.test/account/1" || registered.ContactEmail != "admin@example.test" || registered.AccountStatus != "valid" {
		t.Fatalf("unexpected existing register response: %#v", registered)
	}
}

func TestCreateCertificateACMEOrderPersistsOrderStatus(t *testing.T) {
	var baseURL string
	var receivedOrderEnvelope struct {
		Protected string `json:"protected"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	var receivedAuthorizationEnvelope struct {
		Protected string `json:"protected"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	acmeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Replay-Nonce", "order-test-nonce")
			w.WriteHeader(http.StatusNoContent)
		case "/new-account":
			w.Header().Set("Location", baseURL+"/account/1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"valid"}`))
		case "/new-order":
			if r.Method != http.MethodPost {
				t.Fatalf("new order method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&receivedOrderEnvelope); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Location", baseURL+"/order/1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"status":"pending",
				"authorizations":["` + baseURL + `/authz/1","` + baseURL + `/authz/2"],
				"finalize":"` + baseURL + `/finalize/1"
			}`))
		case "/authz/1":
			if r.Method != http.MethodPost {
				t.Fatalf("authorization method = %s, want POST", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&receivedAuthorizationEnvelope); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"identifier":{"type":"dns","value":"example.com"},
				"status":"pending",
				"challenges":[{"type":"dns-01","url":"` + baseURL + `/challenge/1","status":"pending","token":"token-one"}]
			}`))
		case "/authz/2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"identifier":{"type":"dns","value":"www.example.com"},
				"status":"pending",
				"challenges":[{"type":"dns-01","url":"` + baseURL + `/challenge/2","status":"pending","token":"token-two"}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer acmeServer.Close()
	baseURL = acmeServer.URL
	handler, _ := newTestHandlerWithDirectory(t, acmeServer.URL+"/directory")
	cookie := registerAdmin(t, handler)
	createDNSRec := requestWithCookie(handler, http.MethodPost, "/api/v1/dns-accounts", `{
		"name":"AliDNS 主账号",
		"provider":"alidns",
		"accessKey":"AKIDEXAMPLE1234567890",
		"secretKey":"VerySecretValueShouldNotLeak"
	}`, cookie)
	if createDNSRec.Code != http.StatusCreated {
		t.Fatalf("create dns account = %d %s", createDNSRec.Code, createDNSRec.Body.String())
	}
	var dnsAccount DNSAccountDTO
	if err := json.Unmarshal(createDNSRec.Body.Bytes(), &dnsAccount); err != nil {
		t.Fatal(err)
	}
	createCertificateRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications", `{
		"primaryDomain":"example.com",
		"sans":["www.example.com"],
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
	withoutAccountRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications/"+application.ID+"/acme/order", "", cookie)
	if withoutAccountRec.Code != http.StatusConflict {
		t.Fatalf("order without acme account = %d %s, want 409", withoutAccountRec.Code, withoutAccountRec.Body.String())
	}
	registerRec := requestWithCookie(handler, http.MethodPost, "/api/v1/acme/account/register", "{}", cookie)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register acme account = %d %s", registerRec.Code, registerRec.Body.String())
	}

	orderRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications/"+application.ID+"/acme/order", "", cookie)
	if orderRec.Code != http.StatusOK {
		t.Fatalf("create acme order = %d %s", orderRec.Code, orderRec.Body.String())
	}
	if strings.Contains(orderRec.Body.String(), "PRIVATE KEY") || strings.Contains(orderRec.Body.String(), "secretKey") {
		t.Fatalf("order response leaked sensitive data: %s", orderRec.Body.String())
	}
	var ordered CertificateApplicationDTO
	if err := json.Unmarshal(orderRec.Body.Bytes(), &ordered); err != nil {
		t.Fatal(err)
	}
	if ordered.Status != certificateOrdered || ordered.OrderStatus != "pending" || ordered.OrderURL != acmeServer.URL+"/order/1" || ordered.FinalizeURL != acmeServer.URL+"/finalize/1" {
		t.Fatalf("unexpected ordered dto: %#v", ordered)
	}
	if len(ordered.AuthorizationURLs) != 2 || ordered.AuthorizationURLs[0] != acmeServer.URL+"/authz/1" || ordered.AuthorizationURLs[1] != acmeServer.URL+"/authz/2" {
		t.Fatalf("unexpected ordered authz urls: %#v", ordered.AuthorizationURLs)
	}
	protectedJSON, err := base64.RawURLEncoding.DecodeString(receivedOrderEnvelope.Protected)
	if err != nil {
		t.Fatal(err)
	}
	protected := string(protectedJSON)
	if !strings.Contains(protected, `"kid":"`+acmeServer.URL+`/account/1"`) || strings.Contains(protected, `"jwk"`) {
		t.Fatalf("unexpected order protected header: %s", protected)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(receivedOrderEnvelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(payloadJSON)
	if !strings.Contains(payload, `"value":"example.com"`) || !strings.Contains(payload, `"value":"www.example.com"`) {
		t.Fatalf("unexpected order payload: %s", payload)
	}
	authorizationsRec := requestWithCookie(handler, http.MethodGet, "/api/v1/certificates/applications/"+application.ID+"/acme/authorizations", "", cookie)
	if authorizationsRec.Code != http.StatusOK {
		t.Fatalf("list acme authorizations = %d %s", authorizationsRec.Code, authorizationsRec.Body.String())
	}
	var authorizations []CertificateAuthorizationDTO
	if err := json.Unmarshal(authorizationsRec.Body.Bytes(), &authorizations); err != nil {
		t.Fatal(err)
	}
	if len(authorizations) != 2 {
		t.Fatalf("authorization count = %d, want 2: %#v", len(authorizations), authorizations)
	}
	if authorizations[0].Domain != "example.com" || authorizations[0].DNS01 == nil || authorizations[0].DNS01.RecordName != "_acme-challenge.example.com" || authorizations[0].DNS01.RecordType != "TXT" || authorizations[0].DNS01.RecordValue == "" {
		t.Fatalf("unexpected first authorization: %#v", authorizations[0])
	}
	if authorizations[1].Domain != "www.example.com" || authorizations[1].DNS01 == nil || authorizations[1].DNS01.RecordName != "_acme-challenge.www.example.com" || authorizations[1].DNS01.RecordValue == "" {
		t.Fatalf("unexpected second authorization: %#v", authorizations[1])
	}
	authzProtectedJSON, err := base64.RawURLEncoding.DecodeString(receivedAuthorizationEnvelope.Protected)
	if err != nil {
		t.Fatal(err)
	}
	authzProtected := string(authzProtectedJSON)
	if !strings.Contains(authzProtected, `"kid":"`+acmeServer.URL+`/account/1"`) || strings.Contains(authzProtected, `"jwk"`) {
		t.Fatalf("unexpected authorization protected header: %s", authzProtected)
	}
	if receivedAuthorizationEnvelope.Payload != "" {
		t.Fatalf("authorization POST-as-GET payload = %q, want empty", receivedAuthorizationEnvelope.Payload)
	}
}

func TestProtectedDNSAndCertificateApplicationFlow(t *testing.T) {
	handler, dataFile := newTestHandlerWithData(t)

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
	data, err := os.ReadFile(dataFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "VerySecretValueShouldNotLeak") || !strings.Contains(string(data), "enc:v1:") {
		t.Fatalf("dns secret should be encrypted in data file: %s", string(data))
	}

	updateDNSRec := requestWithCookie(handler, http.MethodPatch, "/api/v1/dns-accounts/"+dnsAccount.ID, `{
		"name":"AliDNS 轮换账号",
		"provider":"dnspod",
		"accessKey":"ROTATEDAKID123456",
		"secretKey":"RotatedSecretValueShouldNotLeak",
		"remark":"已轮换"
	}`, cookie)
	if updateDNSRec.Code != http.StatusOK {
		t.Fatalf("update dns account = %d %s", updateDNSRec.Code, updateDNSRec.Body.String())
	}
	if strings.Contains(updateDNSRec.Body.String(), "RotatedSecretValueShouldNotLeak") || strings.Contains(updateDNSRec.Body.String(), "secretKey") {
		t.Fatalf("dns update dto leaked secret: %s", updateDNSRec.Body.String())
	}
	if err := json.Unmarshal(updateDNSRec.Body.Bytes(), &dnsAccount); err != nil {
		t.Fatal(err)
	}
	if dnsAccount.Name != "AliDNS 轮换账号" || dnsAccount.Provider != "dnspod" || dnsAccount.Remark != "已轮换" {
		t.Fatalf("unexpected updated dns dto: %#v", dnsAccount)
	}
	data, err = os.ReadFile(dataFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "RotatedSecretValueShouldNotLeak") || !strings.Contains(string(data), "enc:v1:") {
		t.Fatalf("rotated dns secret should be encrypted in data file: %s", string(data))
	}

	precheckRec := requestWithCookie(handler, http.MethodPost, "/api/v1/certificates/applications/precheck", `{
		"primaryDomain":"Example.COM.",
		"sans":["www.example.com","*.example.com","www.example.com"],
		"dnsAccountId":"`+dnsAccount.ID+`",
		"challengeMode":"dns-01"
	}`, cookie)
	if precheckRec.Code != http.StatusOK {
		t.Fatalf("precheck certificate application = %d %s", precheckRec.Code, precheckRec.Body.String())
	}
	var precheck CertificatePrecheckDTO
	if err := json.Unmarshal(precheckRec.Body.Bytes(), &precheck); err != nil {
		t.Fatal(err)
	}
	if precheck.PrimaryDomain != "example.com" || precheck.DomainCount != 3 || precheck.DNSAccountName != dnsAccount.Name || precheck.DNSProvider != dnsAccount.Provider {
		t.Fatalf("unexpected precheck dto: %#v", precheck)
	}
	if len(precheck.SANs) != 2 || precheck.SANs[0] != "www.example.com" || precheck.SANs[1] != "*.example.com" {
		t.Fatalf("unexpected precheck sans: %#v", precheck.SANs)
	}
	if len(precheck.Warnings) == 0 {
		t.Fatalf("precheck should include wildcard warning: %#v", precheck)
	}
	listApplicationsAfterPrecheckRec := requestWithCookie(handler, http.MethodGet, "/api/v1/certificates/applications", "", cookie)
	if listApplicationsAfterPrecheckRec.Code != http.StatusOK || strings.TrimSpace(listApplicationsAfterPrecheckRec.Body.String()) != "[]" {
		t.Fatalf("precheck should not create application: %d %s", listApplicationsAfterPrecheckRec.Code, listApplicationsAfterPrecheckRec.Body.String())
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

	deleteCertificateRec := requestWithCookie(handler, http.MethodDelete, "/api/v1/certificates/applications/"+application.ID, "", cookie)
	if deleteCertificateRec.Code != http.StatusNoContent {
		t.Fatalf("delete certificate application = %d %s, want 204", deleteCertificateRec.Code, deleteCertificateRec.Body.String())
	}

	deleteDNSAfterCertificateRec := requestWithCookie(handler, http.MethodDelete, "/api/v1/dns-accounts/"+dnsAccount.ID, "", cookie)
	if deleteDNSAfterCertificateRec.Code != http.StatusNoContent {
		t.Fatalf("delete dns after certificate deletion = %d %s, want 204", deleteDNSAfterCertificateRec.Code, deleteDNSAfterCertificateRec.Body.String())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, _ := newTestHandlerWithData(t)
	return handler
}

func newTestHandlerWithData(t *testing.T) (http.Handler, string) {
	t.Helper()
	return newTestHandlerWithDirectory(t, "https://acme.example.test/directory")
}

func newTestHandlerWithDirectory(t *testing.T, directoryURL string) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "store.json")
	keyFile := filepath.Join(dir, "secret.key")
	acmeKeyFile := filepath.Join(dir, "acme-account.key")
	st, err := store.Open(dataFile)
	if err != nil {
		t.Fatal(err)
	}
	box, err := secretbox.Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	acmeAccountKey, err := acmeaccount.Open(acmeKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	api := New(config.Config{
		FrontendOrigin:     "https://app.example.test",
		SecretKeyFile:      keyFile,
		ACMEAccountKeyFile: acmeKeyFile,
		ACMEDirectoryURL:   directoryURL,
		ACMETermsAgreed:    true,
		SessionTTL:         time.Hour,
	}, st, box, acmeAccountKey, nil)
	return api.Routes(), dataFile
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
