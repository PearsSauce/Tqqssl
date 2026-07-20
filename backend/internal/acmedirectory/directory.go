package acmedirectory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Directory struct {
	NewNonce   string `json:"newNonce"`
	NewAccount string `json:"newAccount"`
	NewOrder   string `json:"newOrder"`
	RevokeCert string `json:"revokeCert,omitempty"`
	KeyChange  string `json:"keyChange,omitempty"`
	Meta       Meta   `json:"meta,omitempty"`
}

type Meta struct {
	TermsOfService          string   `json:"termsOfService,omitempty"`
	Website                 string   `json:"website,omitempty"`
	CAAIdentities           []string `json:"caaIdentities,omitempty"`
	ExternalAccountRequired bool     `json:"externalAccountRequired,omitempty"`
}

type CheckResult struct {
	DirectoryURL            string   `json:"directoryUrl"`
	NewNonce                string   `json:"newNonce"`
	NewAccount              string   `json:"newAccount"`
	NewOrder                string   `json:"newOrder"`
	TermsOfService          string   `json:"termsOfService,omitempty"`
	Website                 string   `json:"website,omitempty"`
	ExternalAccountRequired bool     `json:"externalAccountRequired"`
	Warnings                []string `json:"warnings"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func Check(ctx context.Context, directoryURL string, client HTTPClient) (CheckResult, error) {
	directoryURL = strings.TrimSpace(directoryURL)
	if directoryURL == "" {
		return CheckResult{}, errors.New("ACME directory URL 不能为空")
	}
	parsed, err := url.Parse(directoryURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return CheckResult{}, errors.New("ACME directory URL 格式不正确")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return CheckResult{}, errors.New("ACME directory URL 只支持 http 或 https")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directoryURL, nil)
	if err != nil {
		return CheckResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return CheckResult{}, fmt.Errorf("ACME directory 返回 HTTP %d", resp.StatusCode)
	}
	var directory Directory
	if err := json.NewDecoder(resp.Body).Decode(&directory); err != nil {
		return CheckResult{}, fmt.Errorf("解析 ACME directory 失败: %w", err)
	}
	if strings.TrimSpace(directory.NewNonce) == "" || strings.TrimSpace(directory.NewAccount) == "" || strings.TrimSpace(directory.NewOrder) == "" {
		return CheckResult{}, errors.New("ACME directory 缺少 newNonce、newAccount 或 newOrder")
	}
	result := CheckResult{
		DirectoryURL:            directoryURL,
		NewNonce:                directory.NewNonce,
		NewAccount:              directory.NewAccount,
		NewOrder:                directory.NewOrder,
		TermsOfService:          directory.Meta.TermsOfService,
		Website:                 directory.Meta.Website,
		ExternalAccountRequired: directory.Meta.ExternalAccountRequired,
		Warnings:                warnings(directory),
	}
	return result, nil
}

func warnings(directory Directory) []string {
	values := []string{}
	if strings.TrimSpace(directory.Meta.TermsOfService) == "" {
		values = append(values, "ACME directory 未返回 termsOfService。")
	}
	if directory.Meta.ExternalAccountRequired {
		values = append(values, "该 ACME CA 要求 External Account Binding，当前个人版尚未实现。")
	}
	return values
}
