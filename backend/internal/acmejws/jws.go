package acmejws

import (
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
	"strings"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Envelope struct {
	Protected string `json:"protected"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

func FetchNonce(ctx context.Context, nonceURL string, client HTTPClient) (string, error) {
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

func NewJWKEnvelope(accountKey *acmeaccount.AccountKey, url string, nonce string, payload any) (Envelope, error) {
	privateKey := privateKey(accountKey)
	if privateKey == nil {
		return Envelope{}, errors.New("ACME account key 未加载")
	}
	return newEnvelope(privateKey, map[string]any{
		"alg":   "ES256",
		"nonce": nonce,
		"url":   url,
		"jwk":   jwk(privateKey),
	}, payload)
}

func NewKIDEnvelope(accountKey *acmeaccount.AccountKey, url string, nonce string, kid string, payload any) (Envelope, error) {
	privateKey := privateKey(accountKey)
	if privateKey == nil {
		return Envelope{}, errors.New("ACME account key 未加载")
	}
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return Envelope{}, errors.New("ACME account URL 未注册")
	}
	return newEnvelope(privateKey, map[string]any{
		"alg":   "ES256",
		"nonce": nonce,
		"url":   url,
		"kid":   kid,
	}, payload)
}

func NewKIDPostAsGetEnvelope(accountKey *acmeaccount.AccountKey, url string, nonce string, kid string) (Envelope, error) {
	privateKey := privateKey(accountKey)
	if privateKey == nil {
		return Envelope{}, errors.New("ACME account key 未加载")
	}
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return Envelope{}, errors.New("ACME account URL 未注册")
	}
	return newEnvelopeFromPayload64(privateKey, map[string]any{
		"alg":   "ES256",
		"nonce": nonce,
		"url":   url,
		"kid":   kid,
	}, "")
}

func JWKThumbprint(accountKey *acmeaccount.AccountKey) (string, error) {
	privateKey := privateKey(accountKey)
	if privateKey == nil {
		return "", errors.New("ACME account key 未加载")
	}
	payload := struct {
		Crv string `json:"crv"`
		Kty string `json:"kty"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}{
		Crv: "P-256",
		Kty: "EC",
		X:   base64.RawURLEncoding.EncodeToString(fixedBytes(privateKey.X, 32)),
		Y:   base64.RawURLEncoding.EncodeToString(fixedBytes(privateKey.Y, 32)),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func newEnvelope(privateKey *ecdsa.PrivateKey, protected map[string]any, payload any) (Envelope, error) {
	protectedJSON, err := json.Marshal(protected)
	if err != nil {
		return Envelope{}, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	protected64 := base64.RawURLEncoding.EncodeToString(protectedJSON)
	payload64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return newEnvelopeFromEncoded(privateKey, protected64, payload64)
}

func newEnvelopeFromPayload64(privateKey *ecdsa.PrivateKey, protected map[string]any, payload64 string) (Envelope, error) {
	protectedJSON, err := json.Marshal(protected)
	if err != nil {
		return Envelope{}, err
	}
	protected64 := base64.RawURLEncoding.EncodeToString(protectedJSON)
	return newEnvelopeFromEncoded(privateKey, protected64, payload64)
}

func newEnvelopeFromEncoded(privateKey *ecdsa.PrivateKey, protected64 string, payload64 string) (Envelope, error) {
	digest := sha256.Sum256([]byte(protected64 + "." + payload64))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		return Envelope{}, err
	}
	signature := append(fixedBytes(r, 32), fixedBytes(s, 32)...)
	return Envelope{
		Protected: protected64,
		Payload:   payload64,
		Signature: base64.RawURLEncoding.EncodeToString(signature),
	}, nil
}

func privateKey(accountKey *acmeaccount.AccountKey) *ecdsa.PrivateKey {
	if accountKey == nil {
		return nil
	}
	return accountKey.PrivateKey()
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
