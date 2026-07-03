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

// ServerConfigNotFoundError is returned when server.toml is missing at startup.
type ServerConfigNotFoundError struct {
	Path string
}

func (e ServerConfigNotFoundError) Error() string {
	return fmt.Sprintf("No configuration file found at %s. Run 'arx-ca server config init' to generate one.", e.Path)
}

const (
	serverConfigFileName = "server.toml"
	cliConfigDirName     = ".arx-ca"
	cliConfigFileName    = "cli.yaml"
	agentConfigDirName   = ".arx-cert-service"
	agentConfigFileName  = "agent.yaml"
	serverEnvPrefix      = "ARX"
	agentEnvPrefix       = "ARX_AGENT"
)

var (
	activeServerConfig       ServerConfig
	activeCLIConfig          CLIConfig
	activeAgentConfig        AgentConfig
	serverConfigPathOverride string
	agentConfigPathOverride  string
)

// SetServerConfigPath forces server.toml to load from an absolute path on the next InitServerConfig call.
func SetServerConfigPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve server config path: %w", err)
	}
	serverConfigPathOverride = abs
	return nil
}

// ResolveServerConfigPath returns the absolute server.toml path from an explicit flag or the executable directory.
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

// WriteDefaultServerConfig marshals DefaultServerConfig to TOML at path with mode 0600.
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

	defaults, err := DefaultServerConfigForExecutable()
	if err != nil {
		return fmt.Errorf("build default config: %w", err)
	}
	raw, err := marshalTOMLConfig(defaults)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}

// InitServerConfig loads server.toml beside the executable (or from SetServerConfigPath) and binds it to Viper.
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
	v.SetConfigType("toml")
	v.SetEnvPrefix(serverEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	applyServerViperDefaults(v, defaults)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read server config %s: %w", configPath, err)
	}
	var loaded ServerConfig
	if err := unmarshalServerConfig(v, &loaded); err != nil {
		return fmt.Errorf("unmarshal server config: %w", err)
	}

	migrateLegacyAdminPassword(v, &loaded)
	migrateLegacyCABootstrap(v, &loaded)
	migrateLegacyProvisioners(v, &loaded)

	if err := HealServerConfig(configPath, &loaded); err != nil {
		return err
	}

	applyCAPasswordEnvOverride(&loaded)
	if _, ok := os.LookupEnv(caPasswordEnvVar); ok {
		v.Set("ca.password", loaded.CA.Password)
	}

	syncServerSecurityFieldsInViper(v, loaded)

	activeServerConfig = normalizeServerConfig(loaded)
	return nil
}

// ServerConfigPath returns the absolute path to the active server.toml file.
func ServerConfigPath() (string, error) {
	return serverConfigFilePath()
}

func syncServerSecurityFieldsInViper(v *viper.Viper, cfg ServerConfig) {
	v.Set("security.jwt_secret", cfg.Security.JWTSecret)
	v.Set("bootstrap.admin_password", cfg.Bootstrap.AdminPassword)
}

func migrateLegacyAdminPassword(v *viper.Viper, cfg *ServerConfig) {
	if cfg == nil || strings.TrimSpace(cfg.Bootstrap.AdminPassword) != "" {
		return
	}
	if legacy := strings.TrimSpace(v.GetString("security.initial_admin_password")); legacy != "" {
		cfg.Bootstrap.AdminPassword = legacy
		return
	}
	if legacy := strings.TrimSpace(v.GetString("bootstrap.admin_password_hash")); legacy != "" {
		cfg.Bootstrap.AdminPassword = legacy
	}
}

// migrateLegacyCABootstrap loads CABootstrap when server.toml uses the legacy "CABootstrap" key
// instead of the canonical "ca_bootstrap" field expected by Viper/mapstructure.
func migrateLegacyCABootstrap(v *viper.Viper, cfg *ServerConfig) {
	if cfg == nil || v == nil {
		return
	}
	for _, key := range []string{"CABootstrap", "cabootstrap", "ca_bootstrap"} {
		if !v.IsSet(key) {
			continue
		}
		raw := v.GetStringMap(key)
		if len(raw) == 0 {
			continue
		}
		legacy := CABootstrapFromMap(raw)
		cfg.CABootstrap = mergeCABootstrap(cfg.CABootstrap, legacy)
		return
	}
}

func mergeCABootstrap(current, overlay CABootstrapConfig) CABootstrapConfig {
	if strings.TrimSpace(overlay.RootCN) != "" {
		current.RootCN = overlay.RootCN
	}
	if strings.TrimSpace(overlay.IntermediateCN) != "" {
		current.IntermediateCN = overlay.IntermediateCN
	}
	if strings.TrimSpace(overlay.Organization) != "" {
		current.Organization = overlay.Organization
	}
	if strings.TrimSpace(overlay.Country) != "" {
		current.Country = overlay.Country
	}
	if overlay.KeySize > 0 {
		current.KeySize = overlay.KeySize
	}
	return current
}

func cabootstrapConfigEmpty(b CABootstrapConfig) bool {
	return strings.TrimSpace(b.RootCN) == "" &&
		strings.TrimSpace(b.IntermediateCN) == "" &&
		strings.TrimSpace(b.Organization) == "" &&
		strings.TrimSpace(b.Country) == "" &&
		b.KeySize <= 0
}

// InitCLIConfig loads or creates ~/.arx-ca/cli.yaml and binds it to Viper.
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

// ReloadServerConfigFromDisk re-reads server.toml into Viper and the active in-memory snapshot.
func ReloadServerConfigFromDisk(configPath string) error {
	if strings.TrimSpace(configPath) == "" {
		var err error
		configPath, err = serverConfigFilePath()
		if err != nil {
			return err
		}
	}

	v := viper.GetViper()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read server config %s: %w", configPath, err)
	}

	var loaded ServerConfig
	if err := unmarshalServerConfig(v, &loaded); err != nil {
		return fmt.Errorf("unmarshal server config: %w", err)
	}

	migrateLegacyAdminPassword(v, &loaded)
	migrateLegacyCABootstrap(v, &loaded)
	migrateLegacyProvisioners(v, &loaded)
	applyCAPasswordEnvOverride(&loaded)
	syncServerSecurityFieldsInViper(v, loaded)

	activeServerConfig = normalizeServerConfig(loaded)
	return nil
}

// migrateLegacyProvisioners loads ca.provisioners when server.toml uses legacy PascalCase keys.
func migrateLegacyProvisioners(v *viper.Viper, cfg *ServerConfig) {
	if cfg == nil || v == nil {
		return
	}

	for _, key := range []string{"ca.Provisioners", "ca.provisioners"} {
		raw := v.GetStringMap(key)
		if len(raw) == 0 {
			continue
		}
		legacy := provisionersFromLegacyMap(raw)
		cfg.CA.Provisioners = mergeCAProvisionersConfig(cfg.CA.Provisioners, legacy)
		return
	}
}

func provisionersFromLegacyMap(raw map[string]any) CAProvisionersConfig {
	var out CAProvisionersConfig
	if acmeRaw, ok := raw["ACME"]; ok {
		if m, ok := acmeRaw.(map[string]any); ok {
			out.ACME = acmeProvisionerFromLegacyMap(m)
		}
	} else if acmeRaw, ok := raw["acme"]; ok {
		if m, ok := acmeRaw.(map[string]any); ok {
			out.ACME = acmeProvisionerFromLegacyMap(m)
		}
	}
	if scepRaw, ok := raw["SCEP"]; ok {
		if m, ok := scepRaw.(map[string]any); ok {
			out.SCEP = scepProvisionerFromLegacyMap(m)
		}
	} else if scepRaw, ok := raw["scep"]; ok {
		if m, ok := scepRaw.(map[string]any); ok {
			out.SCEP = scepProvisionerFromLegacyMap(m)
		}
	}
	return out
}

func acmeProvisionerFromLegacyMap(raw map[string]any) ACMEProvisionerConfig {
	var out ACMEProvisionerConfig
	if enabled, ok := legacyBoolPtr(raw, "Enabled", "enabled"); ok {
		out.Enabled = enabled
	}
	if requireEAB, ok := legacyBool(raw, "RequireEAB", "require_eab"); ok {
		out.RequireEAB = requireEAB
	}
	if challenges, ok := legacyStringSlice(raw, "Challenges", "challenges"); ok {
		out.Challenges = challenges
	}
	if deviceAttest, ok := legacyBool(raw, "DeviceAttestation", "device_attestation"); ok {
		out.DeviceAttestation = deviceAttest
	}
	return out
}

func scepProvisionerFromLegacyMap(raw map[string]any) SCEPProvisionerConfig {
	var out SCEPProvisionerConfig
	if enabled, ok := legacyBoolPtr(raw, "Enabled", "enabled"); ok {
		out.Enabled = enabled
	}
	if deviceAttest, ok := legacyBool(raw, "DeviceAttestation", "device_attestation"); ok {
		out.DeviceAttestation = deviceAttest
	}
	if challenge, ok := legacyString(raw, "ChallengePassword", "challenge_password"); ok {
		out.ChallengePassword = challenge
	}
	return out
}

func legacyBoolPtr(raw map[string]any, keys ...string) (*bool, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return boolPtr(v), true
		}
	}
	return nil, false
}

func legacyBool(raw map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		}
	}
	return false, false
}

func legacyString(raw map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if s, ok := value.(string); ok {
			return s, true
		}
	}
	return "", false
}

func legacyStringSlice(raw map[string]any, keys ...string) ([]string, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch items := value.(type) {
		case []string:
			return append([]string(nil), items...), true
		case []any:
			out := make([]string, 0, len(items))
			for _, item := range items {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
			return out, true
		}
	}
	return nil, false
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

// SetAgentConfigPath forces agent.yaml to load from an absolute path on the next InitAgentConfig call.
func SetAgentConfigPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve agent config path: %w", err)
	}
	agentConfigPathOverride = abs
	return nil
}

// ResolveAgentConfigPath returns the absolute agent.yaml path from an explicit flag or ~/.arx-cert-service/agent.yaml.
func ResolveAgentConfigPath(configFlag string) (string, error) {
	if strings.TrimSpace(configFlag) != "" {
		return filepath.Abs(configFlag)
	}
	return agentConfigFilePath()
}

// InitAgentConfig loads or creates agent.yaml (never server.toml) and binds it to Viper.
// The path is ~/.arx-cert-service/agent.yaml unless SetAgentConfigPath was called.
func InitAgentConfig() error {
	configPath, err := agentConfigFilePath()
	if err != nil {
		return err
	}

	defaults := DefaultAgentConfig()
	if err := ensureYAMLConfigFile(configPath, defaults, 0o600); err != nil {
		return err
	}

	viper.Reset()
	v := viper.GetViper()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.SetEnvPrefix(agentEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	applyAgentViperDefaults(v, defaults)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read agent config %s: %w", configPath, err)
	}
	if err := unmarshalAgentConfig(v, &activeAgentConfig); err != nil {
		return fmt.Errorf("unmarshal agent config: %w", err)
	}
	activeAgentConfig = normalizeAgentConfig(activeAgentConfig)
	return nil
}

// AgentConfigFromViper returns the active agent configuration after InitAgentConfig.
func AgentConfigFromViper() AgentConfig {
	cfg := activeAgentConfig
	if err := unmarshalAgentConfig(viper.GetViper(), &cfg); err == nil {
		cfg = normalizeAgentConfig(cfg)
	}
	return cfg
}

// ApplyServerRuntimeFromViper exports server.toml values into CA_API_* and OTEL_* when unset.
func ApplyServerRuntimeFromViper() {
	cfg := ServerConfigFromViper()
	setEnvIfEmpty("CA_API_LISTEN_ADDR", cfg.ListenAddress())
	setEnvIfEmpty("CA_API_CA_CONFIG", cfg.CA.ConfigPath())
	if cfg.Database.UsesPostgreSQL() {
		setEnvIfEmpty("CA_API_DB_TYPE", "postgresql")
		setEnvIfEmpty("CA_API_DB_DATA_SOURCE", cfg.Database.DSN())
	}
	setEnvIfEmpty("CA_API_BOOTSTRAP_ADMIN_EMAIL", cfg.Bootstrap.AdminEmail)
	hash := cfg.BootstrapAdminPasswordHash()
	if hash == "" {
		hash = strings.TrimSpace(cfg.Bootstrap.AdminPassword)
	}
	setEnvIfEmpty("CA_API_BOOTSTRAP_ADMIN_PASSWORD_HASH", hash)
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
	applyProvisionerRuntimeEnv(cfg.CA.Provisioners)
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

func agentConfigFilePath() (string, error) {
	if agentConfigPathOverride != "" {
		return agentConfigPathOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, agentConfigDirName, agentConfigFileName), nil
}

func ensureYAMLConfigFile(path string, defaults any, fileMode os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	dirPerm := os.FileMode(0o755)
	if filepath.Base(dir) == cliConfigDirName || filepath.Base(dir) == agentConfigDirName {
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
	m, err := structToMap(v)
	if err != nil {
		return nil, err
	}
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func structToMap(v any) (map[string]any, error) {
	var m map[string]any
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "mapstructure",
		Result:  &m,
	})
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(v); err != nil {
		return nil, err
	}
	return m, nil
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
	v.SetDefault("server.tls.enabled", d.Server.TLS.Enabled)
	v.SetDefault("server.tls.cert_file", d.Server.TLS.CertFile)
	v.SetDefault("server.tls.key_file", d.Server.TLS.KeyFile)
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
	v.SetDefault("ca.max_ttl", d.CA.MaxTTL)
	v.SetDefault("security.token_expiration_hours", d.Security.TokenExpirationHours)
	v.SetDefault("bootstrap.admin_email", d.Bootstrap.AdminEmail)
	v.SetDefault("bootstrap.admin_password", d.Bootstrap.AdminPassword)
	v.SetDefault("telemetry.service_name", d.Telemetry.ServiceName)
	v.SetDefault("telemetry.exporter_endpoint", d.Telemetry.ExporterEndpoint)
	v.SetDefault("telemetry.exporter_insecure", d.Telemetry.ExporterInsecure)
	v.SetDefault("telemetry.sdk_disabled", d.Telemetry.SDKDisabled)
	v.SetDefault("service.run_as_user", d.Service.RunAsUser)
	v.SetDefault("service.install_dir", d.Service.InstallDir)
	v.SetDefault("webui.enabled", d.WebUI.Enabled)
	v.SetDefault("webui.ui_dir", d.WebUI.UIDir)
	v.SetDefault("webui.path_prefix", d.WebUI.PathPrefix)
	v.SetDefault("webui.listen_address", d.WebUI.ListenAddress)
	v.SetDefault("webui.max_body_size", d.WebUI.MaxBodySize)
	v.SetDefault("webui.read_timeout", d.WebUI.ReadTimeout)
	v.SetDefault("webui.write_timeout", d.WebUI.WriteTimeout)
	v.SetDefault("webui.tls.enabled", d.WebUI.TLS.Enabled)
	v.SetDefault("webui.tls.cert_file", d.WebUI.TLS.CertFile)
	v.SetDefault("webui.tls.key_file", d.WebUI.TLS.KeyFile)
	v.SetDefault("security.cookie_same_site", d.Security.CookieSameSite)
	v.SetDefault("webui.cors.allowed_origins", d.WebUI.CORS.AllowedOrigins)
	v.SetDefault("webui.cors.allowed_methods", d.WebUI.CORS.AllowedMethods)
	v.SetDefault("webui.cors.allowed_headers", d.WebUI.CORS.AllowedHeaders)
	v.SetDefault("webui.cors.allow_credentials", d.WebUI.CORS.AllowCredentials)
	v.SetDefault("updater.enabled", d.Updater.Enabled)
	v.SetDefault("updater.channel", d.Updater.Channel)
	v.SetDefault("updater.notify_only", d.Updater.NotifyOnly)
	v.SetDefault("updater.check_interval", d.Updater.CheckInterval)
	v.SetDefault("updater.view_changelog_after_update", d.Updater.ViewChangelogAfterUpdate)
}

func applyCLIViperDefaults(v *viper.Viper, d CLIConfig) {
	v.SetDefault("log_level", d.LogLevel)
}

func unmarshalAgentConfig(v *viper.Viper, cfg *AgentConfig) error {
	return v.Unmarshal(cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
	)))
}

func applyAgentViperDefaults(v *viper.Viper, d AgentConfig) {
	v.SetDefault("daemon.check_interval", d.Daemon.CheckInterval)
	v.SetDefault("daemon.renew_threshold", d.Daemon.RenewThreshold)
}

func normalizeAgentConfig(cfg AgentConfig) AgentConfig {
	def := DefaultAgentConfig()
	if strings.TrimSpace(cfg.Daemon.CheckInterval) == "" {
		cfg.Daemon.CheckInterval = def.Daemon.CheckInterval
	}
	if strings.TrimSpace(cfg.Daemon.RenewThreshold) == "" {
		cfg.Daemon.RenewThreshold = def.Daemon.RenewThreshold
	}
	for i := range cfg.Daemon.ManagedCerts {
		m := &cfg.Daemon.ManagedCerts[i]
		if strings.TrimSpace(m.Protocol) == "" {
			m.Protocol = AgentProtocolAPI
		}
		if m.ProtocolName() == AgentProtocolACME && strings.TrimSpace(m.ChallengeType) == "" {
			m.ChallengeType = AgentChallengeHTTP01
		}
	}
	return cfg
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
	if strings.TrimSpace(cfg.CA.MaxTTL) == "" {
		cfg.CA.MaxTTL = def.CA.MaxTTL
	}

	if cfg.Security.TokenExpirationHours <= 0 {
		cfg.Security.TokenExpirationHours = def.Security.TokenExpirationHours
	}
	if strings.TrimSpace(cfg.Security.CookieSameSite) == "" {
		cfg.Security.CookieSameSite = def.Security.CookieSameSite
	}
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		if v := strings.TrimSpace(os.Getenv("CA_API_JWT_SECRET")); v != "" {
			cfg.Security.JWTSecret = v
		} else if v := strings.TrimSpace(os.Getenv("ARX_SECURITY_JWT_SECRET")); v != "" {
			cfg.Security.JWTSecret = v
		}
	}

	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = def.Telemetry.ServiceName
	}
	if cfg.Telemetry.ExporterEndpoint == "" {
		cfg.Telemetry.ExporterEndpoint = def.Telemetry.ExporterEndpoint
	}

	cfg.Bootstrap = normalizeBootstrap(cfg.Bootstrap)
	cfg.CABootstrap = cfg.EffectiveCABootstrap()
	cfg.CA.Provisioners = cfg.CA.EffectiveProvisioners()
	cfg.WebUI = normalizeWebUI(cfg.WebUI)
	cfg.Updater = normalizeUpdater(cfg.Updater)
	return cfg
}

func normalizeUpdater(u UpdaterConfig) UpdaterConfig {
	def := DefaultServerConfig().Updater
	if strings.TrimSpace(u.Channel) == "" {
		u.Channel = def.Channel
	}
	if strings.TrimSpace(u.CheckInterval) == "" {
		u.CheckInterval = def.CheckInterval
	}
	return u
}

func normalizeWebUI(w WebUIConfig) WebUIConfig {
	def := DefaultServerConfig().WebUI
	if strings.TrimSpace(w.UIDir) == "" {
		w.UIDir = def.UIDir
	}
	if strings.TrimSpace(w.PathPrefix) == "" {
		w.PathPrefix = def.PathPrefix
	}
	if strings.TrimSpace(w.ListenAddress) == "" {
		w.ListenAddress = def.ListenAddress
	}
	if w.MaxBodySize <= 0 {
		w.MaxBodySize = def.MaxBodySize
	}
	if strings.TrimSpace(w.ReadTimeout) == "" {
		w.ReadTimeout = def.ReadTimeout
	}
	if strings.TrimSpace(w.WriteTimeout) == "" {
		w.WriteTimeout = def.WriteTimeout
	}
	if len(w.CORS.AllowedOrigins) == 0 {
		w.CORS.AllowedOrigins = append([]string(nil), def.CORS.AllowedOrigins...)
	}
	if len(w.CORS.AllowedMethods) == 0 {
		w.CORS.AllowedMethods = append([]string(nil), def.CORS.AllowedMethods...)
	}
	if len(w.CORS.AllowedHeaders) == 0 {
		w.CORS.AllowedHeaders = append([]string(nil), def.CORS.AllowedHeaders...)
	}
	normalizeWebUICORS(&w.CORS)
	return w
}

func normalizeBootstrap(b Bootstrap) Bootstrap {
	def := DefaultServerConfig().Bootstrap
	if b.AdminEmail == "" {
		b.AdminEmail = def.AdminEmail
	}
	if b.AdminPassword == "" {
		b.AdminPassword = def.AdminPassword
	}
	if v := strings.TrimSpace(os.Getenv("CA_API_BOOTSTRAP_ADMIN_EMAIL")); v != "" {
		b.AdminEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("CA_API_BOOTSTRAP_ADMIN_PASSWORD_HASH")); v != "" {
		b.AdminPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("ARX_BOOTSTRAP_ADMIN_EMAIL")); v != "" {
		b.AdminEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH")); v != "" {
		b.AdminPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("CA_API_BOOTSTRAP_ADMIN_PASSWORD")); v != "" && !IsBcryptPasswordHash(b.AdminPassword) {
		b.AdminPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("ARX_BOOTSTRAP_ADMIN_PASSWORD")); v != "" && !IsBcryptPasswordHash(b.AdminPassword) {
		b.AdminPassword = v
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
