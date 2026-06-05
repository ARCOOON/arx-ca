package ca

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.step.sm/crypto/pemutil"

	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/smallstep/certificates/acme"
	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"

	_ "github.com/smallstep/certificates/cas/softcas"
)

const (
	engineName         = "step-ca"
	defaultConfigRel   = "config/ca.json"
	defaultResource    = "arx-ca"
	defaultCAAddress   = "127.0.0.1:9443"
	defaultCADNS       = "localhost"
	defaultProvisioner = "ca-admin"
)

// PKIEngine wraps the step-ca SDK and exposes CA lifecycle operations to the API layer.
type PKIEngine struct {
	configPath string
	basePath   string
	config     *authconfig.Config
	auth       *authority.Authority
	password   []byte
	rootPEM    []byte

	acmeDB       acme.DB
	acmeLinker   acme.Linker
	acmeDNS      string
	acmeHandler  http.Handler
	appDB        *sql.DB
	scepHandler  http.Handler
	ndesHandler  http.Handler
	ndesRegistry *NDESRegistry
	baseCtx      context.Context
	templates    *TemplateStore

	appConfig    config.Config
	k8sReviewer  *K8sTokenReviewer
	maxCertTTL   time.Duration
	provisioners config.CAProvisionersConfig
}

// ConfigPath returns the absolute path to ca.json.
func (e *PKIEngine) ConfigPath() string {
	return e.configPath
}

// BasePath returns the PKI storage root directory.
func (e *PKIEngine) BasePath() string {
	return e.basePath
}

// Authority returns the initialized step-ca signing authority.
func (e *PKIEngine) Authority() *authority.Authority {
	return e.auth
}

// RootCertPEM returns the Root CA certificate encoded in PEM format.
func (e *PKIEngine) RootCertPEM() []byte {
	if len(e.rootPEM) == 0 {
		return nil
	}
	out := make([]byte, len(e.rootPEM))
	copy(out, e.rootPEM)
	return out
}

// CAPassword returns the CA master password used for key escrow encryption.
func (e *PKIEngine) CAPassword() string {
	if e == nil || len(e.password) == 0 {
		return ""
	}
	return string(e.password)
}

// IntermediateCertPEM returns the Intermediate CA certificate encoded in PEM format.
func (e *PKIEngine) IntermediateCertPEM() []byte {
	if e == nil || e.auth == nil {
		return nil
	}
	cert := e.auth.GetIntermediateCertificate()
	if cert == nil {
		return nil
	}
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(block)
}

// RootFingerprint returns the SHA-256 fingerprint of the root certificate.
func (e *PKIEngine) RootFingerprint() string {
	cert, err := pemutil.ParseCertificate(e.rootPEM)
	if err != nil || cert == nil {
		return ""
	}
	sum := sha256.Sum256(cert.Raw)
	return strings.ToLower(hex.EncodeToString(sum[:]))
}

// Status returns the current CA backend health snapshot.
func (e *PKIEngine) Status() models.CABackendStatus {
	if e == nil || e.auth == nil || e.config == nil {
		return models.CABackendStatus{
			Status:      "unhealthy",
			Message:     "CA engine is not initialized",
			Engine:      engineName,
			Initialized: false,
		}
	}

	if len(e.rootPEM) == 0 {
		return models.CABackendStatus{
			Status:      "unhealthy",
			Message:     "root certificate is not loaded",
			Engine:      engineName,
			Initialized: false,
		}
	}

	dbStatus := "in-memory"
	if e.config.DB != nil && e.config.DB.Type != "" {
		dbStatus = e.config.DB.Type
	}
	cryptoBackend := string(e.appConfig.Crypto.Backend)
	if cryptoBackend == "" {
		cryptoBackend = string(config.CryptoBackendLocal)
	}

	return models.CABackendStatus{
		Status:      "healthy",
		Message:     fmt.Sprintf("CA operational; crypto=%s; storage=%s; fingerprint=%s", cryptoBackend, dbStatus, e.RootFingerprint()),
		Engine:      engineName,
		Initialized: true,
	}
}

// Shutdown releases step-ca resources.
func (e *PKIEngine) Shutdown() error {
	if e == nil || e.auth == nil {
		return nil
	}
	return e.auth.Shutdown()
}

func resolvePaths(configPath string) (resolvedConfig, basePath string, err error) {
	if strings.TrimSpace(configPath) == "" {
		return "", "", errors.New("config path is required")
	}

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve config path: %w", err)
	}

	info, statErr := os.Stat(absConfig)
	switch {
	case statErr == nil && info.IsDir():
		absConfig = filepath.Join(absConfig, defaultConfigRel)
	case statErr != nil && !os.IsNotExist(statErr):
		return "", "", fmt.Errorf("stat config path: %w", statErr)
	}

	basePath = filepath.Dir(filepath.Dir(absConfig))
	return absConfig, basePath, nil
}

func resolveDBConfig(basePath string) (dbType, dataSource string) {
	if dsn := strings.TrimSpace(os.Getenv("CA_API_DB_DATA_SOURCE")); dsn != "" {
		dbType = strings.TrimSpace(os.Getenv("CA_API_DB_TYPE"))
		if dbType == "" {
			dbType = "postgresql"
		}
		return normalizeDBType(dbType), dsn
	}

	dbType = strings.TrimSpace(os.Getenv("CA_API_DB_TYPE"))
	if dbType == "" {
		dbType = "badgerv2"
	}

	return normalizeDBType(dbType), filepath.Join(basePath, "db")
}

func normalizeDBType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "badger", "badgerv1":
		return "badgerv1"
	case "badgerv2":
		return "badgerv2"
	case "bbolt", "json":
		return "bbolt"
	case "postgresql", "postgres":
		return "postgresql"
	default:
		return "badgerv2"
	}
}

func resolveCAPassword(basePath string) ([]byte, error) {
	if value := strings.TrimSpace(os.Getenv("CA_API_CA_PASSWORD")); value != "" {
		return []byte(value), nil
	}

	passPath := filepath.Join(basePath, "secrets", "password")
	if data, err := os.ReadFile(passPath); err == nil {
		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			return []byte(trimmed), nil
		}
	}

	pass, err := generateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("generate CA password: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(passPath), 0o700); err != nil {
		return nil, fmt.Errorf("create secrets directory: %w", err)
	}
	if err := os.WriteFile(passPath, pass, 0o600); err != nil {
		return nil, fmt.Errorf("write CA password file: %w", err)
	}

	return pass, nil
}

func generateRandomPassword(size int) ([]byte, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return buf, nil
}

func loadRootPEM(cfg *authconfig.Config) ([]byte, error) {
	if cfg == nil || len(cfg.Root) == 0 {
		return nil, errors.New("root certificate path is not configured")
	}

	rootPath := cfg.Root[0]
	cert, err := pemutil.ReadCertificate(rootPath)
	if err != nil {
		return nil, fmt.Errorf("read root certificate %s: %w", rootPath, err)
	}

	return encodeCertificatePEM(cert), nil
}

func encodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
