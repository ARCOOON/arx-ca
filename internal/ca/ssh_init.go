package ca

import (
	"crypto"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/pemutil"
	"golang.org/x/crypto/ssh"

	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"
	"github.com/smallstep/certificates/pki"
)

const (
	sshUserKeyRel    = "secrets/ssh_user_ca_key"
	sshHostKeyRel    = "secrets/ssh_host_ca_key"
	sshUserPublicRel = "certs/ssh_user_ca_key.pub"
	sshHostPublicRel = "certs/ssh_host_ca_key.pub"
)

// SSHEnabled reports whether SSH CA signing keys are configured.
func (e *PKIEngine) SSHEnabled() bool {
	if e == nil || e.config == nil || e.config.SSH == nil {
		return false
	}
	return strings.TrimSpace(e.config.SSH.UserKey) != "" || strings.TrimSpace(e.config.SSH.HostKey) != ""
}

// ensureSSHCA generates SSH signing keys when missing and enables SSH on provisioners.
func ensureSSHCA(configPath, basePath string, password []byte) error {
	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for SSH CA: %w", err)
	}
	if cfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	changed := false

	userKeyPath := filepath.Join(basePath, sshUserKeyRel)
	hostKeyPath := filepath.Join(basePath, sshHostKeyRel)
	userPubPath := filepath.Join(basePath, sshUserPublicRel)
	hostPubPath := filepath.Join(basePath, sshHostPublicRel)

	if !sshKeyMaterialExists(userKeyPath, hostKeyPath, userPubPath, hostPubPath) {
		if err := writeSSHSigningKeys(basePath, password); err != nil {
			return fmt.Errorf("generate SSH signing keys: %w", err)
		}
		changed = true
	}

	if cfg.SSH == nil {
		cfg.SSH = &authconfig.SSHConfig{
			UserKey: sshUserKeyRel,
			HostKey: sshHostKeyRel,
		}
		changed = true
	} else {
		if strings.TrimSpace(cfg.SSH.UserKey) == "" {
			cfg.SSH.UserKey = sshUserKeyRel
			changed = true
		}
		if strings.TrimSpace(cfg.SSH.HostKey) == "" {
			cfg.SSH.HostKey = sshHostKeyRel
			changed = true
		}
	}

	if changed || !sshClaimsEnabled(cfg) {
		if err := enableSSHProvisionerClaims(cfg); err != nil {
			return err
		}
		changed = true
	}

	if !changed {
		return nil
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

func sshKeyMaterialExists(userKey, hostKey, userPub, hostPub string) bool {
	for _, path := range []string{userKey, hostKey, userPub, hostPub} {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func writeSSHSigningKeys(basePath string, password []byte) error {
	for _, dir := range []string{
		filepath.Join(basePath, "secrets"),
		filepath.Join(basePath, "certs"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if err := writeSSHKeyPair(basePath, sshUserKeyRel, sshUserPublicRel, password); err != nil {
		return fmt.Errorf("ssh user CA key: %w", err)
	}
	if err := writeSSHKeyPair(basePath, sshHostKeyRel, sshHostPublicRel, password); err != nil {
		return fmt.Errorf("ssh host CA key: %w", err)
	}

	return nil
}

func writeSSHKeyPair(basePath, privateRel, publicRel string, password []byte) error {
	signer, err := keyutil.GenerateSigner("OKP", "Ed25519", 0)
	if err != nil {
		return fmt.Errorf("generate ed25519 key: %w", err)
	}

	block, err := pemutil.Serialize(signer, pemutil.WithPassword(password))
	if err != nil {
		return fmt.Errorf("serialize private key: %w", err)
	}

	privatePath := filepath.Join(basePath, privateRel)
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	sshPub, err := sshPublicKeyFromSigner(signer)
	if err != nil {
		return err
	}

	publicPath := filepath.Join(basePath, publicRel)
	if err := os.WriteFile(publicPath, ssh.MarshalAuthorizedKey(sshPub), 0o644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}

func sshPublicKeyFromSigner(signer crypto.Signer) (ssh.PublicKey, error) {
	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		return nil, fmt.Errorf("convert public key to ssh format: %w", err)
	}
	return sshPub, nil
}

func sshClaimsEnabled(cfg *authconfig.Config) bool {
	if cfg.AuthorityConfig.Claims != nil && cfg.AuthorityConfig.Claims.EnableSSHCA != nil && *cfg.AuthorityConfig.Claims.EnableSSHCA {
		return true
	}

	for _, prov := range cfg.AuthorityConfig.Provisioners {
		switch p := prov.(type) {
		case *provisioner.JWK:
			if p.Claims != nil && p.Claims.EnableSSHCA != nil && *p.Claims.EnableSSHCA {
				return true
			}
		case *provisioner.OIDC:
			if p.Claims != nil && p.Claims.EnableSSHCA != nil && *p.Claims.EnableSSHCA {
				return true
			}
		}
	}

	return false
}

func enableSSHProvisionerClaims(cfg *authconfig.Config) error {
	enable := true
	if cfg.AuthorityConfig.Claims == nil {
		cfg.AuthorityConfig.Claims = &provisioner.Claims{}
	}
	cfg.AuthorityConfig.Claims.EnableSSHCA = &enable

	for i, prov := range cfg.AuthorityConfig.Provisioners {
		switch p := prov.(type) {
		case *provisioner.JWK:
			if p.Claims == nil {
				p.Claims = &provisioner.Claims{}
			}
			p.Claims.EnableSSHCA = &enable
			cfg.AuthorityConfig.Provisioners[i] = p
		case *provisioner.OIDC:
			if p.Claims == nil {
				p.Claims = &provisioner.Claims{}
			}
			p.Claims.EnableSSHCA = &enable
			cfg.AuthorityConfig.Provisioners[i] = p
		case *provisioner.SSHPOP:
			if p.Claims == nil {
				p.Claims = &provisioner.Claims{}
			}
			p.Claims.EnableSSHCA = &enable
			cfg.AuthorityConfig.Provisioners[i] = p
		}
	}

	return nil
}

// bootstrapSSHKeys generates SSH keys during initial PKI bootstrap via step-ca pki builder.
func bootstrapSSHKeys(p *pki.PKI, password []byte) error {
	if err := p.GenerateSSHSigningKeys(password); err != nil {
		return fmt.Errorf("generate SSH signing keys: %w", err)
	}
	return nil
}
