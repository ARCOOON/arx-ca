package config

import (
	"context"
	"os"
	"strings"
)

// VaultRAConfig holds optional HashiCorp Vault Registration Authority (RA) mode settings (placeholder).
type VaultRAConfig struct {
	Enabled    bool
	Address    string
	Namespace  string
	Role       string
	AuthMethod string
	TokenFile  string
	PKIPath    string
}

// LoadVaultFromEnv reads Vault RA integration environment variables.
func LoadVaultFromEnv() VaultRAConfig {
	enabled := strings.EqualFold(os.Getenv("CA_API_VAULT_ENABLED"), "true") ||
		strings.TrimSpace(os.Getenv("CA_API_VAULT_ADDRESS")) != ""
	return VaultRAConfig{
		Enabled:    enabled,
		Address:    strings.TrimSpace(os.Getenv("CA_API_VAULT_ADDRESS")),
		Namespace:  strings.TrimSpace(os.Getenv("CA_API_VAULT_NAMESPACE")),
		Role:       strings.TrimSpace(os.Getenv("CA_API_VAULT_ROLE")),
		AuthMethod: strings.TrimSpace(os.Getenv("CA_API_VAULT_AUTH_METHOD")),
		TokenFile:  strings.TrimSpace(os.Getenv("CA_API_VAULT_TOKEN_FILE")),
		PKIPath:    strings.TrimSpace(os.Getenv("CA_API_VAULT_PKI_PATH")),
	}
}

// VaultRAClient is the interface a future Vault RA signing integration would implement.
type VaultRAClient interface {
	Healthy(ctx context.Context) error
	SignCertificate(ctx context.Context, csrPEM []byte, ttl string) ([]byte, error)
}

// StubVaultRAClient is a no-op placeholder until Vault RA mode is implemented.
type StubVaultRAClient struct{}

func (StubVaultRAClient) Healthy(context.Context) error {
	return ErrIntegrationDisabled
}

func (StubVaultRAClient) SignCertificate(context.Context, []byte, string) ([]byte, error) {
	return nil, ErrIntegrationDisabled
}

// NewVaultRAClient returns a client implementation based on configuration.
func NewVaultRAClient(cfg VaultRAConfig) VaultRAClient {
	if !cfg.Enabled {
		return StubVaultRAClient{}
	}
	return StubVaultRAClient{}
}
