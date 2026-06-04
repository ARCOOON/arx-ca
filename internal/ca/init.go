package ca

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallstep/certificates/authority"
	authconfig "github.com/smallstep/certificates/authority/config"
	"github.com/smallstep/certificates/cas/apiv1"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/certificates/pki"
	"github.com/smallstep/cli-utils/step"

	"github.com/ARCOOON/arx-ca/internal/config"

	_ "github.com/smallstep/certificates/cas/softcas"
)

// InitCA initializes or loads a local Root CA and Intermediate CA using the step-ca SDK.
// configPath must point to ca.json or to the PKI base directory containing config/ca.json.
// When the .pki tree is absent, new certificates are generated using cabootstrap from server.yaml.
func InitCA(configPath string, caCfg config.CAConfig, caBootstrap config.CABootstrapConfig) (*PKIEngine, error) {
	maxCertTTL, err := caCfg.MaxTTLDuration()
	if err != nil {
		return nil, err
	}

	provisioners := caCfg.EffectiveProvisioners()
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
		if err := bootstrapPKI(resolvedConfig, basePath, password, appCfg, caBootstrap, provisioners); err != nil {
			return nil, fmt.Errorf("bootstrap PKI: %w", err)
		}
	}

	if err := ensureKMSConfig(resolvedConfig, appCfg); err != nil {
		return nil, fmt.Errorf("configure KMS: %w", err)
	}

	if err := syncCAProvisioners(resolvedConfig, basePath, password, provisioners); err != nil {
		return nil, fmt.Errorf("sync CA provisioners: %w", err)
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

	if err := ensureCRLConfig(resolvedConfig); err != nil {
		return nil, fmt.Errorf("configure CRL: %w", err)
	}

	if err := healPKIConfigPaths(resolvedConfig, basePath); err != nil {
		return nil, fmt.Errorf("heal PKI config paths: %w", err)
	}

	if err := syncCAConfigMaxTTL(resolvedConfig, maxCertTTL); err != nil {
		return nil, fmt.Errorf("sync CA max TTL: %w", err)
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
		configPath:   resolvedConfig,
		basePath:     basePath,
		config:       cfg,
		auth:         authInstance,
		password:     password,
		rootPEM:      rootPEM,
		templates:    templateStore,
		appConfig:    appCfg,
		k8sReviewer:  k8sReviewer,
		maxCertTTL:   maxCertTTL,
		provisioners: provisioners,
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

func bootstrapPKI(
	configPath, basePath string,
	password []byte,
	appCfg config.Config,
	boot config.CABootstrapConfig,
	prov config.CAProvisionersConfig,
) error {
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

	provisionerName := defaultProvisioner
	if name := strings.TrimSpace(os.Getenv("ARX_CA_PROVISIONER_NAME")); name != "" {
		provisionerName = name
	}

	pkiOpts := []pki.Option{
		pki.WithAddress(defaultCAAddress),
		pki.WithDNSNames([]string{defaultCADNS}),
		pki.WithProvisioner(provisionerName),
		pki.WithSSH(),
		pki.WithDeploymentType(pki.StandaloneDeployment),
	}
	if prov.ACMEEnabled() {
		pkiOpts = append(pkiOpts, pki.WithACME())
	}

	p, err := pki.New(
		apiv1.Options{
			Type:      apiv1.SoftCAS,
			IsCreator: true,
		},
		pkiOpts...,
	)
	if err != nil {
		return fmt.Errorf("create PKI builder: %w", err)
	}

	if err := p.GenerateKeyPairs(password); err != nil {
		return fmt.Errorf("generate provisioner keys: %w", err)
	}

	creator, err := bootstrapCACreator()
	if err != nil {
		return err
	}

	root, err := generateBootstrapRoot(p, creator, boot, defaultResource, password)
	if err != nil {
		return fmt.Errorf("generate root certificate: %w", err)
	}

	if err := generateBootstrapIntermediate(p, creator, boot, defaultResource, root, password); err != nil {
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
