package ca

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"
)

// ensureCRLConfig enables CRL generation in ca.json when it is not already configured.
func ensureCRLConfig(configPath string) error {
	if strings.EqualFold(os.Getenv("CA_API_CRL_DISABLED"), "true") {
		return nil
	}

	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for CRL: %w", err)
	}
	if cfg.CRL != nil && cfg.CRL.Enabled {
		return nil
	}

	cfg.CRL = &authconfig.CRLConfig{
		Enabled:          true,
		GenerateOnRevoke: true,
		CacheDuration:    authconfig.DefaultCRLCacheDuration,
		RenewPeriod: &provisioner.Duration{
			Duration: authconfig.DefaultCRLCacheDuration.Duration / 2,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated CA configuration: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("write updated CA configuration: %w", err)
	}

	return nil
}
