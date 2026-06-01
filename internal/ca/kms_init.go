package ca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/pki"
	kmsapi "go.step.sm/crypto/kms/apiv1"

	"github.com/your-org/arx-ca/internal/config"

	_ "go.step.sm/crypto/kms/pkcs11"
)

// ensureKMSConfig updates ca.json when a hardware or cloud KMS backend is enabled.
// Local SoftCAS remains the default when KMS_TYPE is unset or "local".
func ensureKMSConfig(configPath string, cfg config.Config) error {
	if err := cfg.KMS.Validate(); err != nil {
		return err
	}
	if !cfg.KMS.PKCS11Enabled() {
		return nil
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read CA configuration: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse CA configuration: %w", err)
	}

	kmsURI := cfg.KMS.PKCS11KMSURI()
	doc["kms"] = map[string]any{
		"type": string(kmsapi.PKCS11),
		"uri":  kmsURI,
	}
	if pin := strings.TrimSpace(cfg.KMS.PIN); pin != "" {
		doc["kms"].(map[string]any)["pin"] = pin
	}

	if keyRef := cfg.KMS.IntermediateKeyRef(); keyRef != "" {
		doc["key"] = keyRef
	}

	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal CA configuration: %w", err)
	}
	updated = append(updated, '\n')

	if err := os.WriteFile(configPath, updated, 0o600); err != nil {
		return fmt.Errorf("write CA configuration: %w", err)
	}

	return nil
}

// applyKMSBootstrapOptions configures the PKI builder when bootstrapping with PKCS #11.
func applyKMSBootstrapOptions(cfg config.Config) ([]pki.ConfigOption, error) {
	if !cfg.KMS.PKCS11Enabled() {
		return nil, nil
	}
	if err := cfg.KMS.Validate(); err != nil {
		return nil, err
	}
	return []pki.ConfigOption{
		func(c *authconfig.Config) error {
			c.KMS = &kmsapi.Options{
				Type: kmsapi.PKCS11,
				URI:  cfg.KMS.PKCS11KMSURI(),
				Pin:  cfg.KMS.PIN,
			}
			if keyRef := cfg.KMS.IntermediateKeyRef(); keyRef != "" {
				c.IntermediateKey = keyRef
			}
			return nil
		},
	}, nil
}
