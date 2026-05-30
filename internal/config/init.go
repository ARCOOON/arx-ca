package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	serverConfigFileName = "server.yaml"
	cliConfigDirName     = ".arx"
	cliConfigFileName    = "cli.yaml"
)

var (
	activeServerConfig ServerConfig
	activeCLIConfig    CLIConfig
)

// InitServerConfig loads or creates server.yaml beside the executable and binds it to Viper.
func InitServerConfig() error {
	configPath, err := serverConfigFilePath()
	if err != nil {
		return err
	}

	defaults := DefaultServerConfig()
	if err := ensureYAMLConfigFile(configPath, defaults, 0o644); err != nil {
		return err
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	applyServerViperDefaults(viper.GetViper(), defaults)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read server config %s: %w", configPath, err)
	}
	if err := viper.Unmarshal(&activeServerConfig); err != nil {
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
	if err := viper.Unmarshal(&cfg); err == nil {
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
	setEnvIfEmpty("CA_API_CA_CONFIG", cfg.CAConfigPath)
	setEnvIfEmpty("CA_API_DB_TYPE", cfg.DBType)
	setEnvIfEmpty("CA_API_DB_DATA_SOURCE", cfg.DBDataSource)
	setEnvIfEmpty("OTEL_SERVICE_NAME", cfg.OTELServiceName)
	setEnvIfEmpty("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.OTELExporterEndpoint)
	if cfg.OTELExporterInsecure {
		setEnvIfEmpty("OTEL_EXPORTER_OTLP_INSECURE", "true")
	}
	if cfg.OTELSDKDisabled {
		setEnvIfEmpty("OTEL_SDK_DISABLED", "true")
	}
}

func serverConfigFilePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlinks: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), serverConfigFileName), nil
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

	raw, err := yaml.Marshal(defaults)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, raw, fileMode); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}

func applyServerViperDefaults(v *viper.Viper, d ServerConfig) {
	v.SetDefault("host", d.Host)
	v.SetDefault("port", d.Port)
	v.SetDefault("ca_config_path", d.CAConfigPath)
	v.SetDefault("log_level", d.LogLevel)
	v.SetDefault("db_type", d.DBType)
	v.SetDefault("db_data_source", d.DBDataSource)
	v.SetDefault("otel_service_name", d.OTELServiceName)
	v.SetDefault("otel_exporter_endpoint", d.OTELExporterEndpoint)
	v.SetDefault("otel_exporter_insecure", d.OTELExporterInsecure)
	v.SetDefault("otel_sdk_disabled", d.OTELSDKDisabled)
}

func applyCLIViperDefaults(v *viper.Viper, d CLIConfig) {
	v.SetDefault("server_url", d.ServerURL)
	v.SetDefault("log_level", d.LogLevel)
}

func normalizeServerConfig(cfg ServerConfig) ServerConfig {
	def := DefaultServerConfig()
	if cfg.Port <= 0 {
		cfg.Port = def.Port
	}
	if cfg.CAConfigPath == "" {
		cfg.CAConfigPath = def.CAConfigPath
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = def.LogLevel
	}
	if cfg.DBType == "" {
		cfg.DBType = def.DBType
	}
	if cfg.OTELServiceName == "" {
		cfg.OTELServiceName = def.OTELServiceName
	}
	if cfg.OTELExporterEndpoint == "" {
		cfg.OTELExporterEndpoint = def.OTELExporterEndpoint
	}
	return cfg
}

func normalizeCLIConfig(cfg CLIConfig) CLIConfig {
	def := DefaultCLIConfig()
	if cfg.ServerURL == "" {
		cfg.ServerURL = def.ServerURL
	}
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
