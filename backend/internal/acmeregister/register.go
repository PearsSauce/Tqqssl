package acmeregister

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
	"github.com/PearsSauce/Tqqssl/backend/internal/acmedirectory"
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

type jwsEnvelope struct {
	Protected string `json:"protected"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
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
	nonce, err := getNonce(ctx, directory.NewNonce, client)
	if err != nil {
		return Result{}, err
	}
	envelope, err := newAccountJWS(req.AccountKey, directory.NewAccount, nonce, contactEmail)
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

func getNonce(ctx context.Context, nonceURL string, client HTTPClient) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, nonceURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("ACME newNonce 返回 HTTP %d", resp.StatusCode)
	}
	nonce := strings.TrimSpace(resp.Header.Get("Replay-Nonce"))
	if nonce == "" {
		return "", errors.New("ACME newNonce 响应缺少 Replay-Nonce")
	}
	return nonce, nil
}

func newAccountJWS(accountKey *acmeaccount.AccountKey, accountURL string, nonce string, contactEmail string) (jwsEnvelope, error) {
	privateKey := accountKey.PrivateKey()
	if privateKey == nil {
		return jwsEnvelope{}, errors.New("ACME account key 未加载")
	}
	protected := map[string]any{
		"alg":   "ES256",
		"nonce": nonce,
		"url":   accountURL,
		"jwk":   jwk(privateKey),
	}
	payload := directoryPayload{
		TermsOfServiceAgreed: true,
		Contact:              []string{"mailto:" + contactEmail},
	}
	protectedJSON, err := json.Marshal(protected)
	if err != nil {
		return jwsEnvelope{}, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return jwsEnvelope{}, err
	}
	protected64 := base64.RawURLEncoding.EncodeToString(protectedJSON)
	payload64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256.Sum256([]byte(protected64 + "." + payload64))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		return jwsEnvelope{}, err
	}
	signature := append(fixedBytes(r, 32), fixedBytes(s, 32)...)
	return jwsEnvelope{
		Protected: protected64,
		Payload:   payload64,
		Signature: base64.RawURLEncoding.EncodeToString(signature),
	}, nil
}

func jwk(privateKey *ecdsa.PrivateKey) map[string]string {
	return map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(fixedBytes(privateKey.X, 32)),
		"y":   base64.RawURLEncoding.EncodeToString(fixedBytes(privateKey.Y, 32)),
	}
}

func fixedBytes(value *big.Int, size int) []byte {
	raw := value.Bytes()
	if len(raw) >= size {
		return raw[len(raw)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(raw):], raw)
	return out
}
