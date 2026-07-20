package acmeorder

import (
	"bytes"
	"context"
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
	DirectoryURL string
	AccountURL   string
	AccountKey   *acmeaccount.AccountKey
	Domains      []string
}

type Result struct {
	OrderURL          string
	Status            string
	AuthorizationURLs []string
	FinalizeURL       string
	Expires           string
}

type identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type newOrderPayload struct {
	Identifiers []identifier `json:"identifiers"`
}

func Create(ctx context.Context, req Request, client HTTPClient) (Result, error) {
	directoryURL := strings.TrimSpace(req.DirectoryURL)
	accountURL := strings.TrimSpace(req.AccountURL)
	domains := normalizeDomains(req.Domains)
	if directoryURL == "" {
		return Result{}, errors.New("ACME directory URL 未配置")
	}
	if accountURL == "" {
		return Result{}, errors.New("ACME account URL 未注册")
	}
	if req.AccountKey == nil {
		return Result{}, errors.New("ACME account key 未加载")
	}
	if len(domains) == 0 {
		return Result{}, errors.New("ACME order 至少需要一个域名")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	directory, err := acmedirectory.Check(ctx, directoryURL, client)
	if err != nil {
		return Result{}, err
	}
	nonce, err := acmejws.FetchNonce(ctx, directory.NewNonce, client)
	if err != nil {
		return Result{}, err
	}
	envelope, err := acmejws.NewKIDEnvelope(req.AccountKey, directory.NewOrder, nonce, accountURL, newOrderPayload{Identifiers: identifiers(domains)})
	if err != nil {
		return Result{}, err
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return Result{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, directory.NewOrder, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/jose+json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("ACME newOrder 返回 HTTP %d", resp.StatusCode)
	}
	orderURL := strings.TrimSpace(resp.Header.Get("Location"))
	if orderURL == "" {
		return Result{}, errors.New("ACME newOrder 响应缺少 Location")
	}
	var payload struct {
		Status         string   `json:"status"`
		Authorizations []string `json:"authorizations"`
		Finalize       string   `json:"finalize"`
		Expires        string   `json:"expires"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Result{}, fmt.Errorf("解析 ACME newOrder 响应失败: %w", err)
	}
	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "pending"
	}
	return Result{
		OrderURL:          orderURL,
		Status:            status,
		AuthorizationURLs: trimNonEmpty(payload.Authorizations),
		FinalizeURL:       strings.TrimSpace(payload.Finalize),
		Expires:           strings.TrimSpace(payload.Expires),
	}, nil
}

func normalizeDomains(values []string) []string {
	domains := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		domain := strings.ToLower(strings.TrimSpace(value))
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	return domains
}

func identifiers(domains []string) []identifier {
	values := make([]identifier, 0, len(domains))
	for _, domain := range domains {
		values = append(values, identifier{Type: "dns", Value: domain})
	}
	return values
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
