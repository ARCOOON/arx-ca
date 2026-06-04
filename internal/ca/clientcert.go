package ca

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.step.sm/crypto/pemutil"
)

var (
	// ErrNoClientCertificate indicates the TLS connection did not present a client certificate.
	ErrNoClientCertificate = errors.New("no client certificate presented")
	// ErrInvalidClientCertificate indicates the presented client certificate failed validation.
	ErrInvalidClientCertificate = errors.New("invalid client certificate")
)

// ClientCertValidator verifies client certificates issued by this CA.
type ClientCertValidator struct {
	roots         *x509.CertPool
	intermediates *x509.CertPool
	engine        *PKIEngine
}

// NewClientCertValidator builds a validator using the CA root and intermediate certificates.
func NewClientCertValidator(engine *PKIEngine) (*ClientCertValidator, error) {
	if engine == nil || engine.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	roots := x509.NewCertPool()
	rootPEM := engine.RootCertPEM()
	if len(rootPEM) == 0 {
		return nil, errors.New("root CA certificate is not available")
	}
	rootCert, err := pemutil.ParseCertificate(rootPEM)
	if err != nil {
		return nil, fmt.Errorf("parse root CA certificate: %w", err)
	}
	roots.AddCert(rootCert)

	intermediates := x509.NewCertPool()
	if intermediatePEM := engine.IntermediateCertPEM(); len(intermediatePEM) > 0 {
		block, _ := pem.Decode(intermediatePEM)
		if block != nil {
			if intermediateCert, parseErr := x509.ParseCertificate(block.Bytes); parseErr == nil {
				intermediates.AddCert(intermediateCert)
			}
		}
	}

	return &ClientCertValidator{
		roots:         roots,
		intermediates: intermediates,
		engine:        engine,
	}, nil
}

// Validate checks that cert is a valid, non-revoked certificate issued by this CA.
func (v *ClientCertValidator) Validate(cert *x509.Certificate) error {
	if cert == nil {
		return ErrInvalidClientCertificate
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("%w: certificate is not yet valid", ErrInvalidClientCertificate)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("%w: certificate has expired", ErrInvalidClientCertificate)
	}

	opts := x509.VerifyOptions{
		Roots:         v.roots,
		Intermediates: v.intermediates,
		CurrentTime:   now,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClientCertificate, err)
	}

	serial := cert.SerialNumber.String()
	if revoked, _ := v.engine.auth.IsRevoked(serial); revoked {
		return fmt.Errorf("%w: certificate is revoked", ErrInvalidClientCertificate)
	}

	return nil
}

// ClientCertificateFromRequest extracts and validates the leaf client certificate from an mTLS
// connection or from X-Forwarded-Client-Cert when the request was proxied over loopback.
func (v *ClientCertValidator) ClientCertificateFromRequest(r *http.Request) (*x509.Certificate, error) {
	if r == nil {
		return nil, ErrNoClientCertificate
	}

	var cert *x509.Certificate
	var err error

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cert = r.TLS.PeerCertificates[0]
	} else {
		cert, err = forwardedClientCertFromRequest(r)
		if err != nil {
			return nil, err
		}
	}

	if err := v.Validate(cert); err != nil {
		return nil, err
	}
	return cert, nil
}

// CertificateCommonName returns the trimmed subject common name from a certificate.
func CertificateCommonName(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	return strings.TrimSpace(cert.Subject.CommonName)
}
