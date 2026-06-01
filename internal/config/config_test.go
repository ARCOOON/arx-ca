package config

import (
	"strings"
	"testing"
)

func TestLoadFromEnvDefaultsLocal(t *testing.T) {
	t.Setenv(EnvKMSType, "")
	t.Setenv("CA_API_AWS_ENABLED", "")
	t.Setenv("CA_API_GCP_ENABLED", "")
	t.Setenv("CA_API_VAULT_ENABLED", "")

	cfg := LoadFromEnv()
	if !cfg.IsLocalCrypto() {
		t.Fatalf("expected local crypto, got %q", cfg.Crypto.Backend)
	}
	if cfg.Store.Backend != StorageBackendLocal {
		t.Fatalf("expected local storage, got %q", cfg.Store.Backend)
	}
}

func TestLoadKMSPKCS11RequiresModule(t *testing.T) {
	cfg := KMSConfig{Type: KMSTypePKCS11}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error without module path")
	}
	cfg.ModulePath = "/usr/lib/libpkcs11.so"
	cfg.IntermediateKeyURI = "pkcs11:id=1"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPKCS11KMSURI(t *testing.T) {
	cfg := KMSConfig{
		ModulePath: "/opt/lib.so",
		TokenLabel: "arx",
		PIN:        "secret",
	}
	uri := cfg.PKCS11KMSURI()
	if uri == "" {
		t.Fatal("expected non-empty pkcs11 uri")
	}
	if !strings.Contains(uri, "module-path") || !strings.Contains(uri, "pin-value") {
		t.Fatalf("unexpected uri: %s", uri)
	}
}
