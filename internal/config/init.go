package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ServerConfigNotFoundError is returned when server.yaml is missing at startup.
type ServerConfigNotFoundError struct {
	Path string
}

func (e ServerConfigNotFoundError) Error() string {
	return fmt.Sprintf("No configuration file found at %s. Run 'arx server config init' to generate one.", e.Path)
}

const (
	serverConfigFileName = "server.yaml"
	cliConfigDirName     = ".arx"
	cliConfigFileName    = "cli.yaml"
	serverEnvPrefix      = "ARX"
)

var (
	activeServerConfig       ServerConfig
	activeCLIConfig          CLIConfig
	serverConfigPathOverride string
)

// SetServerConfigPath forces server.yaml to load from an absolute path on the next InitServerConfig call.
func SetServerConfigPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve server config path: %w", err)
	}
	serverConfigPathOverride = abs
	return nil
}

// ResolveServerConfigPath returns the absolute server.yaml path from an explicit flag or the executable directory.
func ResolveServerConfigPath(configFlag string) (string, error) {
	if strings.TrimSpace(configFlag) != "" {
		return filepath.Abs(configFlag)
	}
	return serverConfigFilePath()
}

// ExecutablePath returns the absolute path of the current binary, with symlinks resolved.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlinks: %w", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable absolute path: %w", err)
	}
	return abs, nil
}

// WriteDefaultServerConfig marshals DefaultServerConfig to YAML at path with mode 0600.
func WriteDefaultServerConfig(path string, force bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve server config path: %w", err)
	}
	path = abs

	if _, err := os.Stat(path); err == nil {
		if !force {
			return fmt.Errorf("Configuration already exists at %s. Use --force to overwrite.", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	defaults := DefaultServerConfig()
	raw, err := marshalYAMLConfig(defaults)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}

// InitServerConfig loads server.yaml beside the executable (or from SetServerConfigPath) and binds it to Viper.
func InitServerConfig() error {
	configPath, err := serverConfigFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return ServerConfigNotFoundError{Path: configPath}
		}
		return fmt.Errorf("stat server config %s: %w", configPath, err)
	}

	defaults := DefaultServerConfig()

	viper.Reset()
	v := viper.GetViper()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.SetEnvPrefix(serverEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	applyServerViperDefaults(v, defaults)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read server config %s: %w", configPath, err)
	}
	if err := unmarshalServerConfig(v, &activeServerConfig); err != nil {
		return fmt.Errorf("unmarshal server config: %w", err)
	}
	activeServerConfig = normalizeServerConfig(activeServerConfig)
	return nil
}

// InitCLIConfig loads or creates ~/.arx/cli.yaml and binds it to Viper.
func InitCLIConfig() error {
	configPath, err := cliConfigFilePath()
	if err != nil {
		return err
	}

	defaults := DefaultCLIConfig()
	if err := ensureYAMLConfigFile(configPath, defaults, 0o600); err != nil {
		return err
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	applyCLIViperDefaults(viper.GetViper(), defaults)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read CLI config %s: %w", configPath, err)
	}
	if err := viper.Unmarshal(&activeCLIConfig); err != nil {
		return fmt.Errorf("unmarshal CLI config: %w", err)
	}
	activeCLIConfig = normalizeCLIConfig(activeCLIConfig)
	return nil
}

// ServerConfigFromViper returns the active server configuration after InitServerConfig.
func ServerConfigFromViper() ServerConfig {
	cfg := activeServerConfig
	if err := unmarshalServerConfig(viper.GetViper(), &cfg); err == nil {
		cfg = normalizeServerConfig(cfg)
	}
	return cfg
}

// CLIConfigFromViper returns the active CLI configuration after InitCLIConfig.
func CLIConfigFromViper() CLIConfig {
	cfg := activeCLIConfig
	if err := viper.Unmarshal(&cfg); err == nil {
		cfg = normalizeCLIConfig(cfg)
	}
	return cfg
}

// ApplyServerRuntimeFromViper exports server.yaml values into CA_API_* and OTEL_* when unset.
func ApplyServerRuntimeFromViper() {
	cfg := ServerConfigFromViper()
	setEnvIfEmpty("CA_API_LISTEN_ADDR", cfg.ListenAddress())
	setEnvIfEmpty("CA_API_CA_CONFIG", cfg.CA.ConfigPath())
	if cfg.Database.UsesPostgreSQL() {
		setEnvIfEmpty("CA_API_DB_TYPE", "postgresql")
		setEnvIfEmpty("CA_API_DB_DATA_SOURCE", cfg.Database.DSN())
	}
	setEnvIfEmpty("CA_API_BOOTSTRAP_ADMIN_EMAIL", cfg.Bootstrap.AdminEmail)
	setEnvIfEmpty("CA_API_BOOTSTRAP_ADMIN_PASSWORD_HASH", cfg.Bootstrap.AdminPasswordHash)
	if secret := strings.TrimSpace(cfg.Security.JWTSecret); secret != "" {
		setEnvIfEmpty("CA_API_JWT_SECRET", secret)
	}
	setEnvIfEmpty("CA_API_JWT_EXPIRY", cfg.Security.TokenExpiration().String())
	if stepURL := strings.TrimSpace(cfg.CA.StepCAURL); stepURL != "" {
		setEnvIfEmpty("ARX_CA_STEPCA_URL", stepURL)
	}
	if prov := strings.TrimSpace(cfg.CA.ProvisionerName); prov != "" {
		setEnvIfEmpty("ARX_CA_PROVISIONER_NAME", prov)
	}
	caPassword := ResolveSecret(cfg.CA.Password, cfg.CA.PasswordFile)
	if caPassword == "" {
		caPassword = ResolveSecret("", cfg.CA.ProvisionerPasswordFile)
	}
	setEnvIfEmpty("CA_API_CA_PASSWORD", caPassword)
	setEnvIfEmpty("OTEL_SERVICE_NAME", cfg.Telemetry.ServiceName)
	setEnvIfEmpty("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.Telemetry.ExporterEndpoint)
	if cfg.Telemetry.ExporterInsecure {
		setEnvIfEmpty("OTEL_EXPORTER_OTLP_INSECURE", "true")
	}
	if cfg.Telemetry.SDKDisabled {
		setEnvIfEmpty("OTEL_SDK_DISABLED", "true")
	}
}

func serverConfigFilePath() (string, error) {
	if serverConfigPathOverride != "" {
		return serverConfigPathOverride, nil
	}
	exe, err := ExecutablePath()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(exe), serverConfigFileName)
	return filepath.Abs(path)
}

func serverConfigDirectory() (string, error) {
	path, err := serverConfigFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func cliConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, cliConfigDirName, cliConfigFileName), nil
}

func ensureYAMLConfigFile(path string, defaults any, fileMode os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	dirPerm := os.FileMode(0o755)
	if filepath.Base(dir) == cliConfigDirName {
		dirPerm = 0o700
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	raw, err := marshalYAMLConfig(defaults)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, raw, fileMode); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}

func marshalYAMLConfig(v any) ([]byte, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func unmarshalServerConfig(v *viper.Viper, cfg *ServerConfig) error {
	return v.Unmarshal(cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
	)))
}

func applyServerViperDefaults(v *viper.Viper, d ServerConfig) {
	v.SetDefault("server.host", d.Server.Host)
	v.SetDefault("server.port", d.Server.Port)
	v.SetDefault("server.log_level", d.Server.LogLevel)
	v.SetDefault("server.read_timeout", d.Server.ReadTimeout)
	v.SetDefault("server.write_timeout", d.Server.WriteTimeout)
	v.SetDefault("database.driver", d.Database.Driver)
	v.SetDefault("database.path", d.Database.Path)
	v.SetDefault("database.host", d.Database.Host)
	v.SetDefault("database.port", d.Database.Port)
	v.SetDefault("database.user", d.Database.User)
	v.SetDefault("database.dbname", d.Database.DBName)
	v.SetDefault("database.sslmode", d.Database.SSLMode)
	v.SetDefault("database.max_open_conns", d.Database.MaxOpenConns)
	v.SetDefault("database.max_idle_conns", d.Database.MaxIdleConns)
	v.SetDefault("ca.root_path", d.CA.RootPath)
	v.SetDefault("ca.intermediate_path", d.CA.IntermediatePath)
	v.SetDefault("ca.provisioner_name", d.CA.ProvisionerName)
	v.SetDefault("security.token_expiration_hours", d.Security.TokenExpirationHours)
	v.SetDefault("bootstrap.admin_email", d.Bootstrap.AdminEmail)
	v.SetDefault("bootstrap.admin_password_hash", d.Bootstrap.AdminPasswordHash)
	v.SetDefault("telemetry.service_name", d.Telemetry.ServiceName)
	v.SetDefault("telemetry.exporter_endpoint", d.Telemetry.ExporterEndpoint)
	v.SetDefault("telemetry.exporter_insecure", d.Telemetry.ExporterInsecure)
	v.SetDefault("telemetry.sdk_disabled", d.Telemetry.SDKDisabled)
	v.SetDefault("service.run_as_user", d.Service.RunAsUser)
	v.SetDefault("service.install_dir", d.Service.InstallDir)
}

func applyCLIViperDefaults(v *viper.Viper, d CLIConfig) {
	v.SetDefault("log_level", d.LogLevel)
}

func normalizeServerConfig(cfg ServerConfig) ServerConfig {
	def := DefaultServerConfig()

	if cfg.Server.Port <= 0 {
		cfg.Server.Port = def.Server.Port
	}
	if strings.TrimSpace(cfg.Server.Host) == "" {
		cfg.Server.Host = def.Server.Host
	}
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = def.Server.LogLevel
	}
	if cfg.Server.ReadTimeout <= 0 {
		cfg.Server.ReadTimeout = def.Server.ReadTimeout
	}
	if cfg.Server.WriteTimeout <= 0 {
		cfg.Server.WriteTimeout = def.Server.WriteTimeout
	}

	if strings.TrimSpace(cfg.Database.Driver) == "" {
		cfg.Database.Driver = def.Database.Driver
	}
	if strings.TrimSpace(cfg.Database.Path) == "" {
		cfg.Database.Path = def.Database.Path
	}
	if strings.TrimSpace(cfg.Database.Host) == "" {
		cfg.Database.Host = def.Database.Host
	}
	if cfg.Database.Port <= 0 {
		cfg.Database.Port = def.Database.Port
	}
	if strings.TrimSpace(cfg.Database.User) == "" {
		cfg.Database.User = def.Database.User
	}
	if strings.TrimSpace(cfg.Database.DBName) == "" {
		cfg.Database.DBName = def.Database.DBName
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = def.Database.SSLMode
	}
	if cfg.Database.MaxOpenConns <= 0 {
		cfg.Database.MaxOpenConns = def.Database.MaxOpenConns
	}
	if cfg.Database.MaxIdleConns <= 0 {
		cfg.Database.MaxIdleConns = def.Database.MaxIdleConns
	}

	if cfg.CA.ProvisionerName == "" {
		cfg.CA.ProvisionerName = def.CA.ProvisionerName
	}
	if cfg.CA.RootPath == "" {
		cfg.CA.RootPath = def.CA.RootPath
	}
	if cfg.CA.IntermediatePath == "" {
		cfg.CA.IntermediatePath = def.CA.IntermediatePath
	}

	if cfg.Security.TokenExpirationHours <= 0 {
		cfg.Security.TokenExpirationHours = def.Security.TokenExpirationHours
	}
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		if v := strings.TrimSpace(os.Getenv("CA_API_JWT_SECRET")); v != "" {
			cfg.Security.JWTSecret = v
		} else if v := strings.TrimSpace(os.Getenv("ARX_SECURITY_JWT_SECRET")); v != "" {
			cfg.Security.JWTSecret = v
		} else {
			secret, err := GenerateJWTSecret(32)
			if err == nil {
				cfg.Security.JWTSecret = secret
			}
		}
	}

	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = def.Telemetry.ServiceName
	}
	if cfg.Telemetry.ExporterEndpoint == "" {
		cfg.Telemetry.ExporterEndpoint = def.Telemetry.ExporterEndpoint
	}

	cfg.Bootstrap = normalizeBootstrap(cfg.Bootstrap)
	return cfg
}

func normalizeBootstrap(b Bootstrap) Bootstrap {
	def := DefaultServerConfig().Bootstrap
	if b.AdminEmail == "" {
		b.AdminEmail = def.AdminEmail
	}
	if b.AdminPasswordHash == "" {
		b.AdminPasswordHash = def.AdminPasswordHash
	}
	if v := strings.TrimSpace(os.Getenv("CA_API_BOOTSTRAP_ADMIN_EMAIL")); v != "" {
		b.AdminEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("CA_API_BOOTSTRAP_ADMIN_PASSWORD_HASH")); v != "" {
		b.AdminPasswordHash = v
	}
	if v := strings.TrimSpace(os.Getenv("ARX_BOOTSTRAP_ADMIN_EMAIL")); v != "" {
		b.AdminEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH")); v != "" {
		b.AdminPasswordHash = v
	}
	return b
}

func normalizeCLIConfig(cfg CLIConfig) CLIConfig {
	def := DefaultCLIConfig()
	if cfg.LogLevel == "" {
		cfg.LogLevel = def.LogLevel
	}
	return cfg
}

func setEnvIfEmpty(key, value string) {
	if value == "" {
		return
	}
	if os.Getenv(key) == "" {
		_ = os.Setenv(key, value)
	}
}
