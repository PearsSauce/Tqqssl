package acmeregister

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
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
	ContactEmail string
	TermsAgreed  bool
	AccountKey   *acmeaccount.AccountKey
}

type Result struct {
	AccountURL   string
	ContactEmail string
	Status       string
}

type directoryPayload struct {
	TermsOfServiceAgreed bool     `json:"termsOfServiceAgreed"`
	Contact              []string `json:"contact,omitempty"`
}

func Register(ctx context.Context, req Request, client HTTPClient) (Result, error) {
	directoryURL := strings.TrimSpace(req.DirectoryURL)
	contactEmail := strings.ToLower(strings.TrimSpace(req.ContactEmail))
	if directoryURL == "" {
		return Result{}, errors.New("ACME directory URL 未配置")
	}
	if contactEmail == "" {
		return Result{}, errors.New("ACME 联系邮箱不能为空")
	}
	if _, err := mail.ParseAddress(contactEmail); err != nil {
		return Result{}, errors.New("ACME 联系邮箱格式不正确")
	}
	if !req.TermsAgreed {
		return Result{}, errors.New("需要先确认 ACME 服务条款")
	}
	if req.AccountKey == nil {
		return Result{}, errors.New("ACME account key 未加载")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	directory, err := acmedirectory.Check(ctx, directoryURL, client)
	if err != nil {
		return Result{}, err
	}
	if directory.ExternalAccountRequired {
		return Result{}, errors.New("该 ACME CA 要求 External Account Binding，当前个人版尚未实现")
	}
	nonce, err := acmejws.FetchNonce(ctx, directory.NewNonce, client)
	if err != nil {
		return Result{}, err
	}
	envelope, err := acmejws.NewJWKEnvelope(req.AccountKey, directory.NewAccount, nonce, directoryPayload{
		TermsOfServiceAgreed: true,
		Contact:              []string{"mailto:" + contactEmail},
	})
	if err != nil {
		return Result{}, err
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return Result{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, directory.NewAccount, bytes.NewReader(body))
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
		return Result{}, fmt.Errorf("ACME newAccount 返回 HTTP %d", resp.StatusCode)
	}
	accountURL := strings.TrimSpace(resp.Header.Get("Location"))
	if accountURL == "" {
		return Result{}, errors.New("ACME newAccount 响应缺少 Location")
	}
	status := "valid"
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil && strings.TrimSpace(payload.Status) != "" {
		status = strings.TrimSpace(payload.Status)
	}
	return Result{AccountURL: accountURL, ContactEmail: contactEmail, Status: status}, nil
}
