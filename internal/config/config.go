// Package config holds optional integration settings for arx-ca.
// Local PostgreSQL (via step-ca DB config) and local SoftCAS cryptography remain the defaults.
package config

import (
	"os"
	"strings"
)

const (
	// EnvKMSType selects the key management backend (empty or "local" = software keys on disk).
	EnvKMSType = "KMS_TYPE"

	// EnvPKCS11ModulePath is the path to a PKCS #11 module shared library (e.g. YubiKey, SoftHSM, HSM).
	EnvPKCS11ModulePath = "PKCS11_MODULE_PATH"
)

// Config aggregates runtime configuration for optional cloud and hardware integrations.
type Config struct {
	KMS    KMSConfig
	K8s    K8sConfig
	AWS    AWSConfig
	GCP    GCPConfig
	Vault  VaultRAConfig
	Crypto CryptoConfig
	Store  StorageConfig
}

// CryptoConfig describes the active cryptographic backend.
type CryptoConfig struct {
	Backend CryptoBackendType
}

// StorageConfig describes the certificate/metadata persistence backend.
type StorageConfig struct {
	Backend StorageBackendType
	// DSN is used when Backend is PostgreSQL (wired through step-ca ca.json / CA_API_DB_*).
	DSN string
}

// LoadFromEnv builds a Config from environment variables. Unset values keep safe local defaults.
func LoadFromEnv() Config {
	cfg := Config{
		KMS:    LoadKMSFromEnv(),
		K8s:    LoadK8sFromEnv(),
		AWS:    LoadAWSFromEnv(),
		GCP:    LoadGCPFromEnv(),
		Vault:  LoadVaultFromEnv(),
		Crypto: CryptoConfig{Backend: CryptoBackendLocal},
		Store:  StorageConfig{Backend: StorageBackendLocal},
	}

	if dsn := strings.TrimSpace(os.Getenv("CA_API_DB_DATA_SOURCE")); dsn != "" {
		cfg.Store.Backend = StorageBackendPostgreSQL
		cfg.Store.DSN = dsn
	} else if dbType := strings.ToLower(strings.TrimSpace(os.Getenv("CA_API_DB_TYPE"))); dbType == "postgresql" || dbType == "postgres" {
		cfg.Store.Backend = StorageBackendPostgreSQL
	}

	if cfg.KMS.Type == KMSTypePKCS11 {
		cfg.Crypto.Backend = CryptoBackendPKCS11
	} else if cfg.AWS.Enabled {
		cfg.Crypto.Backend = CryptoBackendAWSKMS
	} else if cfg.GCP.Enabled {
		cfg.Crypto.Backend = CryptoBackendGCPKMS
	} else if cfg.Vault.Enabled {
		cfg.Crypto.Backend = CryptoBackendVaultRA
	}

	return cfg
}

// IsLocalCrypto reports whether software keys on disk (SoftCAS) are the active backend.
func (c Config) IsLocalCrypto() bool {
	return c.Crypto.Backend == CryptoBackendLocal || c.Crypto.Backend == ""
}
