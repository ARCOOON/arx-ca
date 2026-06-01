package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"go.step.sm/crypto/pemutil"

	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/authority/provisioner"
	scepAPI "github.com/smallstep/certificates/scep/api"
)

const (
	scepProvisionerName = "scep"
	scepRoutePrefix     = "scep"
)

// SCEPEnabled reports whether the SCEP HTTP handler is configured.
func (e *PKIEngine) SCEPEnabled() bool {
	return e != nil && e.scepHandler != nil
}

// SCEPHandler returns the step-ca SCEP HTTP handler. Mount under /scep.
func (e *PKIEngine) SCEPHandler() http.Handler {
	if e == nil || e.scepHandler == nil {
		return http.NotFoundHandler()
	}
	return e.scepHandler
}

// SCEPProvisionerName returns the default SCEP provisioner name.
func (e *PKIEngine) SCEPProvisionerName() string {
	return scepProvisionerName
}

// SCEPBaseURL returns the local SCEP endpoint URL for the default provisioner.
func (e *PKIEngine) SCEPBaseURL(listenHost string) string {
	host := normalizeListenHost(listenHost)
	if host == "" {
		return ""
	}
	u, err := url.Parse(host)
	if err != nil {
		return ""
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + scepRoutePrefix + "/" + scepProvisionerName
	return u.String()
}

// SCEPChallengeConfigured reports whether a static SCEP challenge password is set.
func (e *PKIEngine) SCEPChallengeConfigured() bool {
	return strings.TrimSpace(os.Getenv("CA_API_SCEP_CHALLENGE")) != ""
}

func (e *PKIEngine) initSCEP() error {
	if e == nil || e.auth == nil {
		return nil
	}
	if e.auth.GetSCEP() == nil {
		return nil
	}

	router := chi.NewRouter()
	scepAPI.Route(router)

	e.scepHandler = newChiProtocolHandler(e, router)
	return nil
}

const (
	scepDecrypterCertRel = "certs/scep_decrypter.crt"
	scepDecrypterKeyRel  = "secrets/scep_decrypter.key"
)

// ensureSCEPProvisioner adds the default SCEP provisioner to ca.json when enabled.
func ensureSCEPProvisioner(configPath, basePath string, password []byte) error {
	if strings.EqualFold(os.Getenv("CA_API_SCEP_DISABLED"), "true") {
		return nil
	}

	cfg, err := authconfig.LoadConfiguration(configPath)
	if err != nil {
		return fmt.Errorf("load configuration for SCEP provisioner: %w", err)
	}
	if cfg.AuthorityConfig == nil {
		return errors.New("authority configuration is missing")
	}

	for _, p := range cfg.AuthorityConfig.Provisioners {
		if p.GetName() == scepProvisionerName && p.GetType() == provisioner.TypeSCEP {
			return nil
		}
	}

	challenge := strings.TrimSpace(os.Getenv("CA_API_SCEP_CHALLENGE"))
	if challenge == "" {
		pass, err := generateRandomPassword(24)
		if err != nil {
			return fmt.Errorf("generate SCEP challenge password: %w", err)
		}
		challenge = string(pass)
		log.Printf("SCEP: generated challenge password; set CA_API_SCEP_CHALLENGE to pin it across restarts")
	}

	decrypterCert, decrypterKey, err := loadOrCreateSCEPDecrypter(basePath, password)
	if err != nil {
		return fmt.Errorf("configure SCEP decrypter: %w", err)
	}

	scepProv := &provisioner.SCEP{
		Type:                   "SCEP",
		Name:                   scepProvisionerName,
		ChallengePassword:      challenge,
		MinimumPublicKeyLength: 2048,
		DecrypterCertificate:   decrypterCert,
		DecrypterKeyPEM:        decrypterKey,
		DecrypterKeyPassword:   string(password),
		Capabilities: []string{
			"SCEPStandard",
			"POSTPKIOperation",
			"SHA-256",
			"AES",
			"DES3",
		},
	}
	if minLen := strings.TrimSpace(os.Getenv("CA_API_SCEP_MIN_KEY_LENGTH")); minLen != "" {
		var n int
		if _, err := fmt.Sscanf(minLen, "%d", &n); err == nil && n > 0 {
			scepProv.MinimumPublicKeyLength = n
		}
	}

	cfg.AuthorityConfig.Provisioners = append(cfg.AuthorityConfig.Provisioners, scepProv)

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

func loadOrCreateSCEPDecrypter(basePath string, password []byte) (certPEM, keyPEM []byte, err error) {
	certPath := filepath.Join(basePath, scepDecrypterCertRel)
	keyPath := filepath.Join(basePath, scepDecrypterKeyRel)

	if certData, readErr := os.ReadFile(certPath); readErr == nil {
		if keyData, keyErr := os.ReadFile(keyPath); keyErr == nil {
			return certData, keyData, nil
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate SCEP decrypter key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generate SCEP decrypter serial: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "arx-ca SCEP decrypter",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return nil, nil, fmt.Errorf("create SCEP decrypter certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse SCEP decrypter certificate: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, nil, err
	}

	certBlock, err := pemutil.Serialize(cert)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, err := pemutil.Serialize(key, pemutil.WithPassword(password))
	if err != nil {
		return nil, nil, err
	}

	certOut := pem.EncodeToMemory(certBlock)
	keyOut := pem.EncodeToMemory(keyBlock)

	if err := os.WriteFile(certPath, certOut, 0o644); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(keyPath, keyOut, 0o600); err != nil {
		return nil, nil, err
	}

	return certOut, keyOut, nil
}
