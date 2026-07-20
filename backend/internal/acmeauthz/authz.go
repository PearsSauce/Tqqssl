package acmeauthz

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
	"github.com/PearsSauce/Tqqssl/backend/internal/acmedirectory"
	"github.com/PearsSauce/Tqqssl/backend/internal/acmejws"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Request struct {
	DirectoryURL      string
	AccountURL        string
	AccountKey        *acmeaccount.AccountKey
	AuthorizationURLs []string
}

type Result struct {
	Authorizations []Authorization
}

type Authorization struct {
	URL      string
	Domain   string
	Wildcard bool
	Status   string
	Expires  string
	DNS01    *DNS01Challenge
}

type DNS01Challenge struct {
	URL              string
	Status           string
	Token            string
	KeyAuthorization string
	RecordName       string
	RecordType       string
	RecordValue      string
}

type authorizationPayload struct {
	Identifier struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"identifier"`
	Status     string             `json:"status"`
	Expires    string             `json:"expires"`
	Wildcard   bool               `json:"wildcard"`
	Challenges []challengePayload `json:"challenges"`
}

type challengePayload struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Status string `json:"status"`
	Token  string `json:"token"`
}

func Inspect(ctx context.Context, req Request, client HTTPClient) (Result, error) {
	directoryURL := strings.TrimSpace(req.DirectoryURL)
	accountURL := strings.TrimSpace(req.AccountURL)
	authorizationURLs := trimNonEmpty(req.AuthorizationURLs)
	if directoryURL == "" {
		return Result{}, errors.New("ACME directory URL 未配置")
	}
	if accountURL == "" {
		return Result{}, errors.New("ACME account URL 未注册")
	}
	if req.AccountKey == nil {
		return Result{}, errors.New("ACME account key 未加载")
	}
	if len(authorizationURLs) == 0 {
		return Result{}, errors.New("ACME order 尚未返回 authorization URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	directory, err := acmedirectory.Check(ctx, directoryURL, client)
	if err != nil {
		return Result{}, err
	}
	thumbprint, err := acmejws.JWKThumbprint(req.AccountKey)
	if err != nil {
		return Result{}, err
	}
	result := Result{Authorizations: make([]Authorization, 0, len(authorizationURLs))}
	for _, authorizationURL := range authorizationURLs {
		authorization, err := fetchAuthorization(ctx, fetchAuthorizationRequest{
			AuthorizationURL: authorizationURL,
			NewNonceURL:      directory.NewNonce,
			AccountURL:       accountURL,
			AccountKey:       req.AccountKey,
			Thumbprint:       thumbprint,
		}, client)
		if err != nil {
			return Result{}, err
		}
		result.Authorizations = append(result.Authorizations, authorization)
	}
	return result, nil
}

type fetchAuthorizationRequest struct {
	AuthorizationURL string
	NewNonceURL      string
	AccountURL       string
	AccountKey       *acmeaccount.AccountKey
	Thumbprint       string
}

func fetchAuthorization(ctx context.Context, req fetchAuthorizationRequest, client HTTPClient) (Authorization, error) {
	nonce, err := acmejws.FetchNonce(ctx, req.NewNonceURL, client)
	if err != nil {
		return Authorization{}, err
	}
	envelope, err := acmejws.NewKIDPostAsGetEnvelope(req.AccountKey, req.AuthorizationURL, nonce, req.AccountURL)
	if err != nil {
		return Authorization{}, err
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return Authorization{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.AuthorizationURL, bytes.NewReader(body))
	if err != nil {
		return Authorization{}, err
	}
	httpReq.Header.Set("Content-Type", "application/jose+json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return Authorization{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Authorization{}, fmt.Errorf("ACME authorization 返回 HTTP %d", resp.StatusCode)
	}
	var payload authorizationPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Authorization{}, fmt.Errorf("解析 ACME authorization 响应失败: %w", err)
	}
	domain := strings.ToLower(strings.TrimSpace(payload.Identifier.Value))
	if domain == "" {
		return Authorization{}, errors.New("ACME authorization 响应缺少 identifier")
	}
	authorization := Authorization{
		URL:      req.AuthorizationURL,
		Domain:   domain,
		Wildcard: payload.Wildcard,
		Status:   strings.TrimSpace(payload.Status),
		Expires:  strings.TrimSpace(payload.Expires),
	}
	if authorization.Status == "" {
		authorization.Status = "pending"
	}
	if dns01 := dns01Challenge(payload.Challenges, domain, req.Thumbprint); dns01 != nil {
		authorization.DNS01 = dns01
	}
	return authorization, nil
}

func dns01Challenge(challenges []challengePayload, domain string, thumbprint string) *DNS01Challenge {
	for _, challenge := range challenges {
		if strings.TrimSpace(challenge.Type) != "dns-01" {
			continue
		}
		token := strings.TrimSpace(challenge.Token)
		if token == "" {
			continue
		}
		keyAuthorization := token + "." + thumbprint
		digest := sha256.Sum256([]byte(keyAuthorization))
		return &DNS01Challenge{
			URL:              strings.TrimSpace(challenge.URL),
			Status:           strings.TrimSpace(challenge.Status),
			Token:            token,
			KeyAuthorization: keyAuthorization,
			RecordName:       dns01RecordName(domain),
			RecordType:       "TXT",
			RecordValue:      base64.RawURLEncoding.EncodeToString(digest[:]),
		}
	}
	return nil
}

func dns01RecordName(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, "*.")))
	return "_acme-challenge." + strings.TrimSuffix(domain, ".")
}

func trimNonEmpty(values []string) []string {
	next := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			next = append(next, value)
		}
	}
	return next
}
