package acmeaccount

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	pemTypeECPrivateKey = "EC PRIVATE KEY"
	keyType             = "ECDSA P-256"
)

type AccountKey struct {
	path string
	key  *ecdsa.PrivateKey
}

func Open(path string) (*AccountKey, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("acme account key file is required")
	}
	key, err := loadOrCreate(path)
	if err != nil {
		return nil, err
	}
	return &AccountKey{path: path, key: key}, nil
}

func (k *AccountKey) Path() string {
	if k == nil {
		return ""
	}
	return k.path
}

func (k *AccountKey) Type() string {
	return keyType
}

func (k *AccountKey) PublicKey() ecdsa.PublicKey {
	if k == nil || k.key == nil {
		return ecdsa.PublicKey{}
	}
	return k.key.PublicKey
}

func (k *AccountKey) PrivateKey() *ecdsa.PrivateKey {
	if k == nil {
		return nil
	}
	return k.key
}

func loadOrCreate(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		if err := writePrivateKey(path, key); err != nil {
			return nil, err
		}
		return key, nil
	}
	if err != nil {
		return nil, err
	}
	return parsePrivateKey(data)
}

func writePrivateKey(path string, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	block := &pem.Block{Type: pemTypeECPrivateKey, Bytes: der}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func parsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, rest := pem.Decode(data)
	if block == nil {
		return nil, errors.New("acme account key file is not PEM encoded")
	}
	if len(strings.TrimSpace(string(rest))) > 0 {
		return nil, errors.New("acme account key file contains trailing data")
	}
	if block.Type != pemTypeECPrivateKey {
		return nil, fmt.Errorf("acme account key PEM type = %q, want %q", block.Type, pemTypeECPrivateKey)
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if key.Curve != elliptic.P256() {
		return nil, errors.New("acme account key must use P-256 curve")
	}
	return key, nil
}
