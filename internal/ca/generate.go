package ca

import (
	"context"
	"crypto"
	"fmt"
	"net"
	"regexp"
	"strings"

	"go.step.sm/crypto/keyutil"

	"github.com/ARCOOON/arx-ca/internal/models"
)

const (
	keyAlgoRSA2048  = "RSA2048"
	keyAlgoECDSA256 = "ECDSA256"
)

var dnsNamePattern = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*$`)

// GenerateCertificate generates a private key, builds a CSR, signs it, and returns PEM material.
// The private key is never persisted by the CA.
func (e *PKIEngine) GenerateCertificate(ctx context.Context, req models.GenerateCertificateRequest) (*models.GenerateCertificateResponse, error) {
	if e == nil || e.auth == nil {
		return nil, fmt.Errorf("CA engine is not initialized")
	}

	cn := strings.TrimSpace(req.CommonName)
	if cn == "" {
		return nil, fmt.Errorf("common_name is required")
	}

	keyAlgo := strings.ToUpper(strings.TrimSpace(req.KeyAlgo))
	if keyAlgo == "" {
		return nil, fmt.Errorf("key_algo is required")
	}

	dnsSANs, ipSANs, err := splitSANEntries(req.SANs)
	if err != nil {
		return nil, err
	}

	sans, err := buildSANs(cn, dnsSANs, ipSANs)
	if err != nil {
		return nil, err
	}

	signer, err := generateSignerForKeyAlgo(keyAlgo)
	if err != nil {
		return nil, err
	}

	subject := subjectInputFromGenerateRequest(req)
	csr, err := createCertificateRequest(cn, sans, subject, signer)
	if err != nil {
		return nil, err
	}

	signOpts, err := e.buildSignOptions(req.TTL)
	if err != nil {
		return nil, err
	}

	requestOpts, err := certificateSignOptions(subject, keyUsageInputFromGenerateRequest(req))
	if err != nil {
		return nil, err
	}

	chain, err := e.signCSR(ctx, csr, signOpts, requestOpts...)
	if err != nil {
		return nil, err
	}

	keyPEM, err := encodePrivateKeyPEM(signer)
	if err != nil {
		return nil, err
	}

	certResp := certificateResponse(chain[0])
	return &models.GenerateCertificateResponse{
		CertificatePEM: certResp.CertificatePEM,
		PrivateKeyPEM:  keyPEM,
	}, nil
}

func generateSignerForKeyAlgo(keyAlgo string) (crypto.Signer, error) {
	switch keyAlgo {
	case keyAlgoRSA2048:
		return keyutil.GenerateSigner("RSA", "", 2048)
	case keyAlgoECDSA256:
		return keyutil.GenerateSigner("EC", "P-256", 0)
	default:
		return nil, fmt.Errorf("unsupported key_algo %q (use %s or %s)", keyAlgo, keyAlgoRSA2048, keyAlgoECDSA256)
	}
}

func splitSANEntries(entries []string) (dnsSANs, ipSANs []string, err error) {
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if ip := net.ParseIP(entry); ip != nil {
			ipSANs = append(ipSANs, entry)
			continue
		}

		if !isValidDNSName(entry) {
			return nil, nil, fmt.Errorf("invalid sans entry %q (expected DNS name or IP address)", entry)
		}
		dnsSANs = append(dnsSANs, entry)
	}
	return dnsSANs, ipSANs, nil
}

func isValidDNSName(name string) bool {
	if name == "" || len(name) > 253 || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	return dnsNamePattern.MatchString(name)
}
