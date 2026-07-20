package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ciphertextPrefix = "enc:v1:"
	keySize          = 32
)

type Box struct {
	key []byte
}

func Open(keyFile string) (*Box, error) {
	keyFile = strings.TrimSpace(keyFile)
	if keyFile == "" {
		return nil, errors.New("secret key file is required")
	}
	key, err := loadOrCreateKey(keyFile)
	if err != nil {
		return nil, err
	}
	return &Box{key: key}, nil
}

func IsCiphertext(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), ciphertextPrefix)
}

func (b *Box) Encrypt(plaintext string) (string, error) {
	if b == nil || len(b.key) != keySize {
		return "", errors.New("secret box is not initialized")
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertextPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (b *Box) Decrypt(ciphertextValue string) (string, error) {
	if b == nil || len(b.key) != keySize {
		return "", errors.New("secret box is not initialized")
	}
	value := strings.TrimSpace(ciphertextValue)
	if !IsCiphertext(value) {
		return "", errors.New("secret value is not encrypted")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, ciphertextPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) <= gcm.NonceSize() {
		return "", errors.New("encrypted secret payload is too short")
	}
	nonce, encrypted := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func loadOrCreateKey(keyFile string) ([]byte, error) {
	data, err := os.ReadFile(keyFile)
	if errors.Is(err, os.ErrNotExist) {
		key := make([]byte, keySize)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(keyFile), 0o700); err != nil {
			return nil, err
		}
		encoded := base64.RawURLEncoding.EncodeToString(key)
		if err := os.WriteFile(keyFile, []byte(encoded+"\n"), 0o600); err != nil {
			return nil, err
		}
		return key, nil
	}
	if err != nil {
		return nil, err
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("decode secret key file: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("secret key file must contain %d bytes, got %d", keySize, len(key))
	}
	return key, nil
}
