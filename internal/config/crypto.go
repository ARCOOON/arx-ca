package config

import "context"

// CryptoBackendType identifies where signing keys are stored.
type CryptoBackendType string

const (
	CryptoBackendLocal   CryptoBackendType = "local"
	CryptoBackendPKCS11  CryptoBackendType = "pkcs11"
	CryptoBackendAWSKMS  CryptoBackendType = "awskms"
	CryptoBackendGCPKMS  CryptoBackendType = "gcpkms"
	CryptoBackendVaultRA CryptoBackendType = "vault-ra"
)

// CryptoBackend abstracts signing key access. LocalCryptoBackend is the default implementation.
type CryptoBackend interface {
	Type() CryptoBackendType
	// Healthy returns nil when the backend is ready to sign.
	Healthy(ctx context.Context) error
}

// LocalCryptoBackend uses SoftCAS (keys on disk under the PKI directory).
type LocalCryptoBackend struct{}

func (LocalCryptoBackend) Type() CryptoBackendType { return CryptoBackendLocal }

func (LocalCryptoBackend) Healthy(context.Context) error { return nil }

// PKCS11CryptoBackend marks hardware/module-backed keys (actual signing is delegated to step-ca KMS).
type PKCS11CryptoBackend struct {
	Config KMSConfig
}

func (b PKCS11CryptoBackend) Type() CryptoBackendType { return CryptoBackendPKCS11 }

func (b PKCS11CryptoBackend) Healthy(ctx context.Context) error {
	if err := b.Config.Validate(); err != nil {
		return err
	}
	return nil
}

// NewCryptoBackend returns the configured CryptoBackend implementation.
func NewCryptoBackend(cfg Config) CryptoBackend {
	switch cfg.Crypto.Backend {
	case CryptoBackendPKCS11:
		return PKCS11CryptoBackend{Config: cfg.KMS}
	default:
		return LocalCryptoBackend{}
	}
}
