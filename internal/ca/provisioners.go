package ca

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/cas/apiv1"
	"github.com/smallstep/certificates/pki"
)

const (
	defaultOIDCProvisionerName = "oidc"
	sshHostKeyRel              = "secrets/ssh_host_ca_key"
	sshUserKeyRel              = "secrets/ssh_user_ca_key"
	sshHostPubRel              = "certs/ssh_host_ca_key.pub"
	sshUserPubRel              = "certs/ssh_user_ca_key.pub"
)

type oidcProvisionerConfig struct {
	Name                  string
	ClientID              string
	ClientSecret          string
	ConfigurationEndpoint string
}

func sshPKIExists(basePath string) bool {
	required := []string{
		filepath.Join(basePath, sshHostKeyRel),
		filepath.Join(basePath, sshUserKeyRel),
		filepath.Join(basePath, sshHostPubRel),
		filepath.Join(basePath, sshUserPubRel),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func ensureSSHCA(configPath, basePath string, password []byte) error {
	cfg, err := authority.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load CA configuration: %w", err)
	}
	if cfg.SSH != nil && sshPKIExists(basePath) {
		return ensureOIDCProvisioner(configPath, cfg)
	}

	if !sshPKIExists(basePath) {
		if err := generateSSHSigningKeys(basePath, password); err != nil {
			return err
		}
	}

	if err := patchCAConfigSSH(configPath, basePath); err != nil {
		return err
	}

	cfg, err = authority.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("reload CA configuration: %w", err)
	}

	return ensureOIDCProvisioner(configPath, cfg)
}

func generateSSHSigningKeys(basePath string, password []byte) error {
	casOptions := apiv1.Options{
		Type: apiv1.SoftCAS,
	}

	p, err := pki.New(casOptions, pki.WithSSH())
	if err != nil {
		return fmt.Errorf("create PKI builder for SSH keys: %w", err)
	}

	if err := p.GenerateSSHSigningKeys(password); err != nil {
		return fmt.Errorf("generate SSH signing keys: %w", err)
	}

	if err := p.WriteFiles(); err != nil {
		return fmt.Errorf("write SSH key files: %w", err)
	}

	_ = basePath
	return nil
}

func patchCAConfigSSH(configPath, basePath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read CA config: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse CA config: %w", err)
	}

	if _, ok := raw["ssh"]; !ok {
		raw["ssh"] = map[string]string{
			"hostKey": sshHostKeyRel,
			"userKey": sshUserKeyRel,
		}
	}

	enableSSHCAOnProvisionerClaims(raw)

	updated, err := json.MarshalIndent(raw, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal CA config: %w", err)
	}
	updated = append(updated, '\n')

	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		return fmt.Errorf("write CA config: %w", err)
	}

	_ = basePath
	return nil
}

func enableSSHCAOnProvisionerClaims(raw map[string]any) {
	authorityBlock, ok := raw["authority"].(map[string]any)
	if !ok {
		return
	}

	provisioners, ok := authorityBlock["provisioners"].([]any)
	if !ok {
		return
	}

	for _, entry := range provisioners {
		prov, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		provType, _ := prov["type"].(string)
		switch strings.ToLower(provType) {
		case "jwk", "oidc", "sshpop":
			claims, ok := prov["claims"].(map[string]any)
			if !ok {
				claims = map[string]any{}
				prov["claims"] = claims
			}
			claims["enableSSHCA"] = true
		}
	}
}

func loadOIDCProvisionerConfigFromEnv() *oidcProvisionerConfig {
	clientID := strings.TrimSpace(os.Getenv("CA_API_OIDC_CLIENT_ID"))
	if clientID == "" {
		return nil
	}

	name := strings.TrimSpace(os.Getenv("CA_API_OIDC_PROVISIONER_NAME"))
	if name == "" {
		name = defaultOIDCProvisionerName
	}

	return &oidcProvisionerConfig{
		Name:                  name,
		ClientID:              clientID,
		ClientSecret:          os.Getenv("CA_API_OIDC_CLIENT_SECRET"),
		ConfigurationEndpoint: strings.TrimSpace(os.Getenv("CA_API_OIDC_CONFIGURATION_ENDPOINT")),
	}
}

func ensureOIDCProvisioner(configPath string, cfg *authconfig.Config) error {
	oidcCfg := loadOIDCProvisionerConfigFromEnv()
	if oidcCfg == nil {
		return nil
	}
	if oidcCfg.ConfigurationEndpoint == "" {
		return errors.New("CA_API_OIDC_CONFIGURATION_ENDPOINT is required when CA_API_OIDC_CLIENT_ID is set")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read CA config: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse CA config: %w", err)
	}

	if oidcProvisionerExists(raw, oidcCfg.ClientID) {
		enableSSHCAOnProvisionerClaims(raw)
		return writeCAConfig(configPath, raw)
	}

	authorityBlock, ok := raw["authority"].(map[string]any)
	if !ok {
		authorityBlock = map[string]any{}
		raw["authority"] = authorityBlock
	}

	provisioners, _ := authorityBlock["provisioners"].([]any)
	provisioners = append(provisioners, map[string]any{
		"type":                  "OIDC",
		"name":                  oidcCfg.Name,
		"clientID":              oidcCfg.ClientID,
		"clientSecret":          oidcCfg.ClientSecret,
		"configurationEndpoint": oidcCfg.ConfigurationEndpoint,
		"claims": map[string]any{
			"enableSSHCA": true,
		},
	})
	authorityBlock["provisioners"] = provisioners

	enableSSHCAOnProvisionerClaims(raw)

	if err := writeCAConfig(configPath, raw); err != nil {
		return err
	}

	_ = cfg
	return nil
}

func oidcProvisionerExists(raw map[string]any, clientID string) bool {
	authorityBlock, ok := raw["authority"].(map[string]any)
	if !ok {
		return false
	}

	provisioners, ok := authorityBlock["provisioners"].([]any)
	if !ok {
		return false
	}

	for _, entry := range provisioners {
		prov, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(provType(prov), "OIDC") {
			if id, _ := prov["clientID"].(string); id == clientID {
				return true
			}
		}
	}

	return false
}

func provType(prov map[string]any) string {
	if t, ok := prov["type"].(string); ok {
		return t
	}
	return ""
}

func writeCAConfig(configPath string, raw map[string]any) error {
	updated, err := json.MarshalIndent(raw, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal CA config: %w", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		return fmt.Errorf("write CA config: %w", err)
	}
	return nil
}
