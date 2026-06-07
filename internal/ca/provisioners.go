package ca

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/ARCOOON/arx-ca/internal/models"
)

const (
	defaultTokenTTL        = 5 * time.Minute
	maxProvisionerTokenTTL = 15 * time.Minute
	oidcProvisionerName    = "oidc-sso"
)

// ListProvisioners returns provisioners configured in the step-ca authority.
func (e *PKIEngine) ListProvisioners(ctx context.Context) (*models.ListProvisionersResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	provisioners, _, err := e.auth.GetProvisioners("", 0)
	if err != nil {
		return nil, fmt.Errorf("list provisioners: %w", err)
	}

	summaries := make([]models.ProvisionerSummary, 0, len(provisioners))
	for _, prov := range provisioners {
		summaries = append(summaries, models.ProvisionerSummary{
			ID:   prov.GetID(),
			Name: prov.GetName(),
			Type: prov.GetType().String(),
		})
	}

	return &models.ListProvisionersResponse{
		Provisioners: summaries,
		Total:        len(summaries),
	}, nil
}

// GenerateProvisionerToken mints a single-use JWK provisioner token for the given subject and SANs.
func (e *PKIEngine) GenerateProvisionerToken(ctx context.Context, req models.ProvisionerTokenRequest) (*models.ProvisionerTokenResponse, error) {
	if e == nil || e.auth == nil {
		return nil, errors.New("CA engine is not initialized")
	}

	cn := strings.TrimSpace(req.CommonName)
	if cn == "" {
		return nil, errors.New("common_name is required")
	}

	sans, err := buildSANs(cn, req.DNSSANs, req.IPSANs)
	if err != nil {
		return nil, err
	}

	provisionerName := strings.TrimSpace(req.Provisioner)
	if provisionerName == "" {
		provisionerName = e.AdminProvisionerName()
	}

	tokenTTL, err := parseProvisionerTokenTTL(req.TokenTTL)
	if err != nil {
		return nil, err
	}

	token, audience, provType, err := e.createProvisionerSignToken(provisionerName, cn, sans, tokenTTL)
	if err != nil {
		return nil, err
	}

	return &models.ProvisionerTokenResponse{
		Token:           token,
		Provisioner:     provisionerName,
		ProvisionerType: provType,
		ExpiresIn:       int(tokenTTL.Seconds()),
		Audience:        audience,
	}, nil
}

func (e *PKIEngine) loadProvisionerByName(name string) (provisioner.Interface, error) {
	if e.auth == nil {
		return nil, errors.New("authority is not initialized")
	}

	prov, err := e.auth.LoadProvisionerByName(name)
	if err != nil {
		return nil, fmt.Errorf("provisioner %q not found", name)
	}
	return prov, nil
}

func (e *PKIEngine) createProvisionerSignToken(provisionerName, subject string, sans []string, tokenTTL time.Duration) (token, audience, provType string, err error) {
	prov, err := e.loadProvisionerByName(provisionerName)
	if err != nil {
		return "", "", "", err
	}

	provType = prov.GetType().String()

	switch p := prov.(type) {
	case *provisioner.JWK:
		kid, encryptedKey, ok := p.GetEncryptedKey()
		if !ok || len(encryptedKey) == 0 {
			return "", "", "", fmt.Errorf("provisioner %q does not have an encrypted signing key", prov.GetName())
		}

		signer, err := decryptProvisionerKey(kid, []byte(encryptedKey), e.password)
		if err != nil {
			return "", "", "", err
		}

		audiences := e.config.GetAudiences().Sign
		if len(audiences) == 0 {
			return "", "", "", errors.New("no sign audiences configured")
		}
		audience = audiences[0]

		token, err = buildProvisionerToken(subject, prov.GetName(), audience, sans, signer, tokenTTL)
		if err != nil {
			return "", "", "", err
		}
		return token, audience, provType, nil
	default:
		return "", "", "", fmt.Errorf("provisioner %q (type %s) cannot mint signing tokens; use a JWK provisioner", prov.GetName(), provType)
	}
}

func parseProvisionerTokenTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTokenTTL, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid token_ttl: %w", err)
	}
	if d <= 0 {
		return 0, errors.New("token_ttl must be greater than zero")
	}
	if d > maxProvisionerTokenTTL {
		return 0, fmt.Errorf("token_ttl must not exceed %s", maxProvisionerTokenTTL)
	}
	return d, nil
}

// ensureAdvancedProvisioners adds optional OIDC provisioner configuration when environment variables are set.
func ensureAdvancedProvisioners(configPath string) error {
	name := strings.TrimSpace(os.Getenv("CA_API_OIDC_NAME"))
	clientID := strings.TrimSpace(os.Getenv("CA_API_OIDC_CLIENT_ID"))
	configEndpoint := strings.TrimSpace(os.Getenv("CA_API_OIDC_CONFIGURATION_ENDPOINT"))

	if name == "" && clientID == "" && configEndpoint == "" {
		return nil
	}

	if name == "" {
		name = oidcProvisionerName
	}
	if clientID == "" || configEndpoint == "" {
		return errors.New("CA_API_OIDC_CLIENT_ID and CA_API_OIDC_CONFIGURATION_ENDPOINT are required when configuring OIDC")
	}

	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for OIDC provisioner: %w", err)
	}
	if cfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	for _, p := range cfg.AuthorityConfig.Provisioners {
		if p.GetName() == name && p.GetType() == provisioner.TypeOIDC {
			return nil
		}
	}

	oidcProv := &provisioner.OIDC{
		Type:                  "OIDC",
		Name:                  name,
		ClientID:              clientID,
		ClientSecret:          os.Getenv("CA_API_OIDC_CLIENT_SECRET"),
		ConfigurationEndpoint: configEndpoint,
	}
	enableSSH := true
	oidcProv.Claims = &provisioner.Claims{
		EnableSSHCA: &enableSSH,
	}
	if tenant := strings.TrimSpace(os.Getenv("CA_API_OIDC_TENANT_ID")); tenant != "" {
		oidcProv.TenantID = tenant
	}
	if domains := strings.TrimSpace(os.Getenv("CA_API_OIDC_DOMAINS")); domains != "" {
		oidcProv.Domains = splitCSV(domains)
	}

	cfg.AuthorityConfig.Provisioners = append(cfg.AuthorityConfig.Provisioners, oidcProv)

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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
