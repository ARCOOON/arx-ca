package ca

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/ARCOOON/arx-ca/internal/config"
)

const defaultK8sProvisionerName = "k8s-sa"

// ensureK8sSAProvisioner registers a Kubernetes Service Account provisioner when enabled.
func ensureK8sSAProvisioner(configPath string, cfg config.K8sConfig) error {
	if !cfg.Enabled {
		return nil
	}

	if len(cfg.PublicKeysPEM) == 0 && !cfg.UsesTokenReviewAPI() {
		return errors.New("CA_API_K8S_ENABLED requires CA_API_K8S_PUBLIC_KEYS(_FILE) or TokenReview API configuration (CA_API_K8S_API_SERVER or in-cluster credentials)")
	}

	// TokenReview-only mode uses the arx-ca reviewer and bridges to the default JWK provisioner.
	if len(cfg.PublicKeysPEM) == 0 {
		return nil
	}

	authCfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for K8s SA provisioner: %w", err)
	}
	if authCfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	name := cfg.Provisioner
	if name == "" {
		name = defaultK8sProvisionerName
	}

	for _, p := range authCfg.AuthorityConfig.Provisioners {
		if p.GetName() == name && p.GetType() == provisioner.TypeK8sSA {
			return nil
		}
	}

	k8sProv := &provisioner.K8sSA{
		Type: "K8sSA",
		Name: name,
	}
	enableSSH := true
	k8sProv.Claims = &provisioner.Claims{
		EnableSSHCA: &enableSSH,
	}
	if len(cfg.PublicKeysPEM) > 0 {
		k8sProv.PubKeys = cfg.PublicKeysPEM
	}

	authCfg.AuthorityConfig.Provisioners = append(authCfg.AuthorityConfig.Provisioners, k8sProv)

	data, err := json.MarshalIndent(authCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated CA configuration: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("write updated CA configuration: %w", err)
	}

	return nil
}

// initK8sReviewer constructs the token reviewer used during enrollment.
func initK8sReviewer(cfg config.K8sConfig) (*K8sTokenReviewer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	return NewK8sTokenReviewer(cfg)
}
