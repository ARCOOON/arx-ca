package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cliapi "github.com/ARCOOON/arx-ca/internal/cli/api"
	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// APIRenewer renews certificates through the Arx CA REST API.
type APIRenewer struct {
	Client *cliapi.Client
}

// NewAPIRenewer builds an API renewer using the shared authenticated HTTP client.
func NewAPIRenewer(client *cliapi.Client) *APIRenewer {
	return &APIRenewer{Client: client}
}

// Renew requests a new certificate from POST /api/v1/certificates/auto and writes PEM files.
func (r *APIRenewer) Renew(ctx context.Context, managed config.ManagedCert) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("API renewer is not configured")
	}

	commonName := strings.TrimSpace(managed.CommonName)
	req := models.AutoCertificateRequest{
		CommonName: commonName,
		DNSSANs:    []string{commonName},
		TemplateID: strings.TrimSpace(managed.Template),
	}

	resp, err := r.Client.AutoCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("request certificate: %w", err)
	}
	if strings.TrimSpace(resp.CertificatePEM) == "" {
		return fmt.Errorf("renewal response did not include a certificate")
	}
	if strings.TrimSpace(resp.PrivateKeyPEM) == "" {
		return fmt.Errorf("renewal response did not include a private key")
	}

	certPath := strings.TrimSpace(managed.CertPath)
	keyPath := strings.TrimSpace(managed.KeyPath)

	if err := writePEMFile(certPath, resp.CertificatePEM); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}
	if err := writePEMFile(keyPath, resp.PrivateKeyPEM); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	return runPostHookIfSet(managed)
}

func runPostHookIfSet(managed config.ManagedCert) error {
	hook := strings.TrimSpace(managed.PostHook)
	if hook == "" {
		return nil
	}
	cmd := exec.Command("sh", "-c", hook)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute hook %q: %w", hook, err)
	}
	return nil
}

func writePEMFile(path, pemContent string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(pemContent), 0o600); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
