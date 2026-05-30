package ca

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/smallstep/cli-utils/step"
	"go.step.sm/crypto/pemutil"

	"github.com/smallstep/certificates/acme"
	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/cas/apiv1"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/certificates/pki"

	"github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/models"

	_ "github.com/smallstep/certificates/cas/softcas"
)

const (
	engineName         = "step-ca"
	defaultConfigRel   = "config/ca.json"
	defaultPKIName     = "Arx CA"
	defaultOrg         = "Arx CA"
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
	scepHandler  http.Handler
	ndesHandler  http.Handler
	ndesRegistry *NDESRegistry
	baseCtx      context.Context
	templates    *TemplateStore

	appConfig   config.Config
	k8sReviewer *K8sTokenReviewer
}

// InitCA initializes or loads a local Root CA and Intermediate CA using the step-ca SDK.
// configPath must point to ca.json or to the PKI base directory containing config/ca.json.
// If the PKI artifacts do not exist, they are generated with ECDSA P-256 keys automatically.
func InitCA(configPath string) (*PKIEngine, error) {
	appCfg := config.LoadFromEnv()
	if err := appCfg.KMS.Validate(); err != nil {
		return nil, fmt.Errorf("kms configuration: %w", err)
	}

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
		if err := bootstrapPKI(resolvedConfig, basePath, password, appCfg); err != nil {
			return nil, fmt.Errorf("bootstrap PKI: %w", err)
		}
	}

	if err := ensureKMSConfig(resolvedConfig, appCfg); err != nil {
		return nil, fmt.Errorf("configure KMS: %w", err)
	}

	if err := ensureACMEProvisioner(resolvedConfig); err != nil {
		return nil, fmt.Errorf("configure ACME provisioner: %w", err)
	}

	if err := ensureAdvancedProvisioners(resolvedConfig); err != nil {
		return nil, fmt.Errorf("configure advanced provisioners: %w", err)
	}

	if err := ensureK8sSAProvisioner(resolvedConfig, appCfg.K8s); err != nil {
		return nil, fmt.Errorf("configure kubernetes service account provisioner: %w", err)
	}

	if err := ensureSSHCA(resolvedConfig, basePath, password); err != nil {
		return nil, fmt.Errorf("configure SSH CA: %w", err)
	}

	if err := ensureSCEPProvisioner(resolvedConfig, basePath, password); err != nil {
		return nil, fmt.Errorf("configure SCEP provisioner: %w", err)
	}

	if err := ensureCRLConfig(resolvedConfig); err != nil {
		return nil, fmt.Errorf("configure CRL: %w", err)
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
	if err != nil && needsBadgerTruncate(err) && cfg.DB != nil {
		switch cfg.DB.Type {
		case "badger", "badgerv1", "badgerv2":
			if healErr := healBadgerDB(cfg.DB.DataSource); healErr != nil {
				return nil, fmt.Errorf("heal badger database: %w", healErr)
			}
			authInstance, err = authority.New(
				cfg,
				authority.WithPassword(password),
				authority.WithQuietInit(),
			)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("initialize step-ca authority: %w", err)
	}

	templateStore, err := newTemplateStore(basePath)
	if err != nil {
		return nil, fmt.Errorf("initialize template store: %w", err)
	}

	k8sReviewer, err := initK8sReviewer(appCfg.K8s)
	if err != nil {
		return nil, fmt.Errorf("initialize kubernetes token reviewer: %w", err)
	}

	engine := &PKIEngine{
		configPath:  resolvedConfig,
		basePath:    basePath,
		config:      cfg,
		auth:        authInstance,
		password:    password,
		rootPEM:     rootPEM,
		templates:   templateStore,
		appConfig:   appCfg,
		k8sReviewer: k8sReviewer,
	}

	if err := engine.initACME(); err != nil {
		return nil, fmt.Errorf("initialize ACME: %w", err)
	}
	if err := engine.initSCEP(); err != nil {
		return nil, fmt.Errorf("initialize SCEP: %w", err)
	}
	if err := engine.initNDES(); err != nil {
		return nil, fmt.Errorf("initialize NDES: %w", err)
	}
	if engine.baseCtx == nil {
		engine.rebuildBaseContext()
	}

	return engine, nil
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

func bootstrapPKI(configPath, basePath string, password []byte, appCfg config.Config) error {
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
		pki.WithACME(),
		pki.WithSSH(),
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

	if err := bootstrapSSHKeys(p, password); err != nil {
		return fmt.Errorf("generate SSH signing keys: %w", err)
	}

	dbType, dbSource := resolveDBConfig(basePath)
	saveOpts := []pki.ConfigOption{
		withConfigPassword(password),
		withDBConfig(dbType, dbSource),
	}
	if kmsOpts, err := applyKMSBootstrapOptions(appCfg); err != nil {
		return err
	} else {
		for _, opt := range kmsOpts {
			saveOpts = append(saveOpts, opt)
		}
	}
	if err := p.Save(saveOpts...); err != nil {
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
