package enroll

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/agent/state"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// Options configures a single-domain enrollment request.
type Options struct {
	Domain string
	TTL    string
}

// Meta describes a successfully enrolled certificate on disk.
type Meta struct {
	Domain string
	Serial string
	Dir    string
}

// AutoCertificate issues a certificate via the admin API (typically POST /api/v1/certificates/auto).
type AutoCertificate func(ctx context.Context, req models.AutoCertificateRequest) (*models.AutoCertificateResponse, error)

// Run requests a certificate for the domain and stores PEM files under
// ~/.arx-cert-service/enrolled/<domain>/.
func Run(ctx context.Context, issue AutoCertificate, opts Options) (*Meta, error) {
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	req := models.AutoCertificateRequest{
		CommonName: domain,
		DNSSANs:    []string{domain},
		TTL:        strings.TrimSpace(opts.TTL),
	}
	resp, err := issue(ctx, req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.CertificatePEM) == "" {
		return nil, fmt.Errorf("enrollment response did not include a certificate")
	}
	if strings.TrimSpace(resp.PrivateKeyPEM) == "" {
		return nil, fmt.Errorf("enrollment response did not include a private key")
	}

	dir, err := enrolledDir(domain)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create enrollment directory: %w", err)
	}

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, []byte(resp.CertificatePEM), 0o600); err != nil {
		return nil, fmt.Errorf("write certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(resp.PrivateKeyPEM), 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	return &Meta{
		Domain: domain,
		Serial: resp.Serial,
		Dir:    dir,
	}, nil
}

func enrolledDir(domain string) (string, error) {
	base, err := state.Dir()
	if err != nil {
		return "", err
	}
	safe := strings.ReplaceAll(domain, string(filepath.Separator), "_")
	return filepath.Join(base, "enrolled", safe), nil
}
