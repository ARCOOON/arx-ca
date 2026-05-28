package ca

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/smallstep/cli-utils/step"
	"go.step.sm/crypto/pemutil"

	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/cas/apiv1"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/certificates/pki"

	"github.com/your-org/ca-api/internal/models"

	_ "github.com/smallstep/certificates/cas/softcas"
)

const (
	engineName        = "step-ca"
	defaultConfigRel  = "config/ca.json"
	defaultPKIName    = "Arx Root CA"
	defaultOrg        = "Arx Root CA"
	defaultResource   = "arx-rootca"
	defaultCAAddress  = "127.0.0.1:9443"
	defaultCADNS      = "localhost"
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
}

// InitCA initializes or loads a local Root CA and Intermediate CA using the step-ca SDK.
// configPath must point to ca.json or to the PKI base directory containing config/ca.json.
// If the PKI artifacts do not exist, they are generated with ECDSA P-256 keys automatically.
func InitCA(configPath string) (*PKIEngine, error) {
	resolvedConfig, basePath, err := resolvePaths(configPath)
	if err != nil {
		return nil, err
	}

	if err := configureStepPath(basePath); err != nil {
		return nil, err
	}

	password, err := resolveCAPassword(basePath)
	if err != nil {
		return nil, err
	}

	if !pkiExists(resolvedConfig, basePath) {
		if err := bootstrapPKI(resolvedConfig, basePath, password); err != nil {
			return nil, fmt.Errorf("bootstrap PKI: %w", err)
		}
	}

	cfg, err := authority.LoadConfiguration(resolvedConfig)
	if err != nil {
		return nil, fmt.Errorf("load CA configuration: %w", err)
	}

	rootPEM, err := loadRootPEM(cfg)
	if err != nil {
		return nil, fmt.Errorf("load root certificate: %w", err)
	}

	authInstance, err := authority.New(
		cfg,
		authority.WithPassword(password),
		authority.WithQuietInit(),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize step-ca authority: %w", err)
	}

	return &PKIEngine{
		configPath: resolvedConfig,
		basePath:   basePath,
		config:     cfg,
		auth:       authInstance,
		password:   password,
		rootPEM:    rootPEM,
	}, nil
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

	return models.CABackendStatus{
		Status:      "healthy",
		Message:     fmt.Sprintf("CA operational; storage=%s; fingerprint=%s", dbStatus, e.RootFingerprint()),
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

func configureStepPath(basePath string) error {
	if err := os.Setenv(step.PathEnv, basePath); err != nil {
		return fmt.Errorf("set %s: %w", step.PathEnv, err)
	}
	if err := step.Init(); err != nil {
		return fmt.Errorf("initialize step environment: %w", err)
	}
	return nil
}

func pkiExists(configPath, basePath string) bool {
	required := []string{
		configPath,
		filepath.Join(basePath, "certs", "root_ca.crt"),
		filepath.Join(basePath, "certs", "intermediate_ca.crt"),
		filepath.Join(basePath, "secrets", "intermediate_ca_key"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func bootstrapPKI(configPath, basePath string, password []byte) error {
	for _, dir := range []string{
		filepath.Join(basePath, "config"),
		filepath.Join(basePath, "certs"),
		filepath.Join(basePath, "secrets"),
		filepath.Join(basePath, "db"),
		filepath.Join(basePath, "templates"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	casOptions := apiv1.Options{
		Type:      apiv1.SoftCAS,
		IsCreator: true,
	}

	p, err := pki.New(
		casOptions,
		pki.WithAddress(defaultCAAddress),
		pki.WithDNSNames([]string{defaultCADNS}),
		pki.WithProvisioner(defaultProvisioner),
		pki.WithDeploymentType(pki.StandaloneDeployment),
	)
	if err != nil {
		return fmt.Errorf("create PKI builder: %w", err)
	}

	if err := p.GenerateKeyPairs(password); err != nil {
		return fmt.Errorf("generate provisioner keys: %w", err)
	}

	root, err := p.GenerateRootCertificate(defaultPKIName, defaultOrg, defaultResource, password)
	if err != nil {
		return fmt.Errorf("generate root certificate: %w", err)
	}

	if err := p.GenerateIntermediateCertificate(defaultPKIName, defaultOrg, defaultResource, root, password); err != nil {
		return fmt.Errorf("generate intermediate certificate: %w", err)
	}

	dbType, dbSource := resolveDBConfig(basePath)
	if err := p.Save(
		withConfigPassword(password),
		withDBConfig(dbType, dbSource),
	); err != nil {
		return fmt.Errorf("persist PKI: %w", err)
	}

	if p.GetCAConfigPath() != configPath {
		if err := copyFile(p.GetCAConfigPath(), configPath); err != nil {
			return fmt.Errorf("align CA config path: %w", err)
		}
	}

	return nil
}

func withConfigPassword(password []byte) pki.ConfigOption {
	return func(cfg *authconfig.Config) error {
		cfg.Password = string(password)
		return nil
	}
}

func withDBConfig(dbType, dataSource string) pki.ConfigOption {
	return func(cfg *authconfig.Config) error {
		cfg.DB = &db.Config{
			Type:       dbType,
			DataSource: dataSource,
		}
		return nil
	}
}

func resolveDBConfig(basePath string) (dbType, dataSource string) {
	dbType = strings.TrimSpace(os.Getenv("CA_API_DB_TYPE"))
	if dbType == "" {
		dbType = "badgerv2"
	}

	switch strings.ToLower(dbType) {
	case "badger", "badgerv1":
		dbType = "badgerv1"
	case "badgerv2":
		dbType = "badgerv2"
	case "bbolt", "json":
		// bbolt is the file-backed alternative when a lightweight store is preferred.
		dbType = "bbolt"
	default:
		dbType = "badgerv2"
	}

	return dbType, filepath.Join(basePath, "db")
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
