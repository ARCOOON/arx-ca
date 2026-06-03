package ca

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.step.sm/crypto/pemutil"

	"github.com/smallstep/certificates/authority"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/your-org/arx-ca/internal/models"
)

// RenewCertificate re-issues a certificate with the same public key and attributes.
func (e *PKIEngine) RenewCertificate(ctx context.Context, certificatePEM, renewToken string) (*models.CertificatePEMResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	cert, token, err := e.resolveRenewCertificate(certificatePEM, renewToken)
	if err != nil {
		return nil, err
	}

	ctx = authority.NewContext(ctx, e.auth)
	if token != "" {
		ctx = authority.NewTokenContext(ctx, token)
	}

	chain, err := e.auth.RenewContext(ctx, cert, nil)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return nil, errors.New("renewal produced an empty certificate chain")
	}

	return certificateResponse(chain[0]), nil
}

// RekeyCertificate re-issues a certificate using a new key from the provided CSR.
func (e *PKIEngine) RekeyCertificate(ctx context.Context, certificatePEM, csrPEM, renewToken string) (*models.CertificatePEMResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	cert, token, err := e.resolveRenewCertificate(certificatePEM, renewToken)
	if err != nil {
		return nil, err
	}

	csr, err := pemutil.ParseCertificateRequest([]byte(strings.TrimSpace(csrPEM)))
	if err != nil {
		return nil, fmt.Errorf("parse certificate signing request: %w", err)
	}

	ctx = authority.NewContext(ctx, e.auth)
	if token != "" {
		ctx = authority.NewTokenContext(ctx, token)
	}

	chain, err := e.auth.RenewContext(ctx, cert, csr.PublicKey)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return nil, errors.New("rekey produced an empty certificate chain")
	}

	return certificateResponse(chain[0]), nil
}

// ResolveRenewTarget resolves the certificate targeted by a renewal request.
func (e *PKIEngine) ResolveRenewTarget(certificatePEM, renewToken string) (*x509.Certificate, error) {
	cert, _, err := e.resolveRenewCertificate(certificatePEM, renewToken)
	return cert, err
}

func (e *PKIEngine) resolveRenewCertificate(certificatePEM, renewToken string) (*x509.Certificate, string, error) {
	renewToken = strings.TrimSpace(renewToken)
	certificatePEM = strings.TrimSpace(certificatePEM)

	switch {
	case renewToken != "":
		ctx := provisioner.NewContextWithMethod(context.Background(), provisioner.RenewMethod)
		ctx = authority.NewContext(ctx, e.auth)
		cert, err := e.auth.AuthorizeRenewToken(ctx, renewToken)
		if err != nil {
			return nil, "", err
		}
		return cert, renewToken, nil
	case certificatePEM != "":
		cert, err := pemutil.ParseCertificate([]byte(certificatePEM))
		if err != nil {
			return nil, "", fmt.Errorf("parse certificate: %w", err)
		}
		return cert, "", nil
	default:
		return nil, "", errors.New("certificate_pem or renew_token is required")
	}
}
