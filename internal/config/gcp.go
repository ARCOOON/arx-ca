package config

import (
	"context"
	"os"
	"strings"
)

// GCPConfig holds optional GCP Cloud KMS / IAM integration settings (placeholder).
type GCPConfig struct {
	Enabled         bool
	ProjectID       string
	Location        string
	KeyRing         string
	CryptoKey       string
	CredentialsFile string
}

// LoadGCPFromEnv reads GCP integration environment variables.
func LoadGCPFromEnv() GCPConfig {
	enabled := strings.EqualFold(os.Getenv("CA_API_GCP_ENABLED"), "true") ||
		strings.TrimSpace(os.Getenv("CA_API_GCP_KMS_KEY")) != ""
	return GCPConfig{
		Enabled:         enabled,
		ProjectID:       strings.TrimSpace(os.Getenv("CA_API_GCP_PROJECT")),
		Location:        strings.TrimSpace(os.Getenv("CA_API_GCP_LOCATION")),
		KeyRing:         strings.TrimSpace(os.Getenv("CA_API_GCP_KEY_RING")),
		CryptoKey:       strings.TrimSpace(os.Getenv("CA_API_GCP_KMS_KEY")),
		CredentialsFile: strings.TrimSpace(os.Getenv("CA_API_GCP_CREDENTIALS_FILE")),
	}
}

// GCPKMSClient is the interface a future GCP Cloud KMS CAS plugin would implement.
type GCPKMSClient interface {
	Healthy(ctx context.Context) error
	SignDigest(ctx context.Context, resourceName string, digest []byte) ([]byte, error)
}

// StubGCPKMSClient is a no-op placeholder until GCP KMS integration is implemented.
type StubGCPKMSClient struct{}

func (StubGCPKMSClient) Healthy(context.Context) error {
	return ErrIntegrationDisabled
}

func (StubGCPKMSClient) SignDigest(context.Context, string, []byte) ([]byte, error) {
	return nil, ErrIntegrationDisabled
}

// NewGCPKMSClient returns a client implementation based on configuration.
func NewGCPKMSClient(cfg GCPConfig) GCPKMSClient {
	if !cfg.Enabled {
		return StubGCPKMSClient{}
	}
	return StubGCPKMSClient{}
}
