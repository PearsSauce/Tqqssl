package certcrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"strings"
)

type CSRBundle struct {
	PrivateKeyPEM string
	CSRPEM        string
}

func GenerateKeyAndCSR(domains []string) (CSRBundle, error) {
	cleanDomains := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		cleanDomains = append(cleanDomains, domain)
	}
	if len(cleanDomains) == 0 {
		return CSRBundle{}, errors.New("CSR 至少需要一个域名")
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return CSRBundle{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return CSRBundle{}, err
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: cleanDomains[0]},
		DNSNames: cleanDomains,
	}, privateKey)
	if err != nil {
		return CSRBundle{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	return CSRBundle{PrivateKeyPEM: string(keyPEM), CSRPEM: string(csrPEM)}, nil
}
