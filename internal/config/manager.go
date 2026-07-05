package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// MaskedSecretValue is returned by the settings API instead of sensitive fields.
const MaskedSecretValue = "***"

// marshalTOMLConfig serializes v as TOML bytes for server.toml persistence.
func marshalTOMLConfig(v any) ([]byte, error) {
	return toml.Marshal(v)
}

// unmarshalTOMLConfig parses TOML bytes into v.
func unmarshalTOMLConfig(data []byte, v any) error {
	return toml.Unmarshal(data, v)
}

// Manager provides concurrent read/write access to server.toml.
type Manager struct {
	mu   sync.RWMutex
	path string
}

// NewManager constructs a Manager for the active server.toml path.
func NewManager() (*Manager, error) {
	path, err := ServerConfigPath()
	if err != nil {
		return nil, err
	}
	return &Manager{path: path}, nil
}

// Path returns the absolute server.toml path managed by this instance.
func (m *Manager) Path() string {
	if m == nil {
		return ""
	}
	return m.path
}

// Get returns the current in-memory server configuration.
func (m *Manager) Get() ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return ServerConfigFromViper()
}

// GetMasked returns the active configuration with sensitive values redacted.
func (m *Manager) GetMasked() ServerConfig {
	return MaskServerConfig(m.Get())
}

// Update merges patch into the active configuration, validates, persists to disk, and reloads memory state.
func (m *Manager) Update(patch ServerConfigPatch) (ServerConfig, error) {
	if m == nil {
		return ServerConfig{}, fmt.Errorf("config manager is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current := ServerConfigFromViper()
	merged, err := mergeServerConfigPatch(current, patch)
	if err != nil {
		return ServerConfig{}, err
	}
	merged = normalizeServerConfig(merged)
	if err := validateServerConfig(merged); err != nil {
		return ServerConfig{}, err
	}
	if err := PersistServerConfig(m.path, merged); err != nil {
		return ServerConfig{}, err
	}
	if err := ReloadServerConfigFromDisk(m.path); err != nil {
		return ServerConfig{}, err
	}
	return MaskServerConfig(merged), nil
}

// ServerConfigPatch carries optional top-level sections for partial PUT updates.
type ServerConfigPatch struct {
	Server      *ServerSettings    `json:"server,omitempty"`
	Database    *DatabaseConfig    `json:"database,omitempty"`
	CA          *CAConfig          `json:"ca,omitempty"`
	CABootstrap *CABootstrapConfig `json:"ca_bootstrap,omitempty"`
	Security    *SecurityConfig    `json:"security,omitempty"`
	Bootstrap   *Bootstrap         `json:"bootstrap,omitempty"`
	Telemetry   *TelemetryConfig   `json:"telemetry,omitempty"`
	Service     *ServiceConfig     `json:"service,omitempty"`
	WebUI       *WebUIConfig       `json:"webui,omitempty"`
	Updater     *UpdaterConfig     `json:"updater,omitempty"`
}

// MaskServerConfig returns a copy of cfg with sensitive values replaced by MaskedSecretValue.
func MaskServerConfig(cfg ServerConfig) ServerConfig {
	out := cfg
	out.Database.Password = maskSecret(cfg.Database.Password)
	out.CA.Password = maskSecret(cfg.CA.Password)
	out.Security.JWTSecret = maskSecret(cfg.Security.JWTSecret)
	out.Bootstrap.AdminPassword = maskBootstrapPassword(cfg.Bootstrap.AdminPassword)
	out.CA.Provisioners.SCEP.ChallengePassword = maskSecret(cfg.CA.Provisioners.SCEP.ChallengePassword)
	return out
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return MaskedSecretValue
}

func maskBootstrapPassword(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if IsBcryptPasswordHash(value) {
		return MaskedSecretValue
	}
	return MaskedSecretValue
}

func mergeServerConfigPatch(current ServerConfig, patch ServerConfigPatch) (ServerConfig, error) {
	out := current

	if patch.Server != nil {
		out.Server = mergeServerSettings(current.Server, *patch.Server)
	}
	if patch.Database != nil {
		out.Database = mergeDatabaseConfig(current.Database, *patch.Database)
	}
	if patch.CA != nil {
		out.CA = mergeCAConfig(current.CA, *patch.CA)
	}
	if patch.CABootstrap != nil {
		out.CABootstrap = mergeCABootstrap(current.CABootstrap, *patch.CABootstrap)
	}
	if patch.Security != nil {
		out.Security = mergeSecurityConfig(current.Security, *patch.Security)
	}
	if patch.Bootstrap != nil {
		out.Bootstrap = mergeBootstrap(current.Bootstrap, *patch.Bootstrap)
	}
	if patch.Telemetry != nil {
		out.Telemetry = *patch.Telemetry
	}
	if patch.Service != nil {
		out.Service = *patch.Service
	}
	if patch.WebUI != nil {
		out.WebUI = mergeWebUIConfig(current.WebUI, *patch.WebUI)
	}
	if patch.Updater != nil {
		out.Updater = *patch.Updater
	}

	return out, nil
}

func mergeServerSettings(current, patch ServerSettings) ServerSettings {
	out := current
	if strings.TrimSpace(patch.Host) != "" {
		out.Host = patch.Host
	}
	if patch.Port > 0 {
		out.Port = patch.Port
	}
	if strings.TrimSpace(patch.LogLevel) != "" {
		out.LogLevel = patch.LogLevel
	}
	if patch.ReadTimeout > 0 {
		out.ReadTimeout = patch.ReadTimeout
	}
	if patch.WriteTimeout > 0 {
		out.WriteTimeout = patch.WriteTimeout
	}
	out.TLS = mergeServerTLSConfig(current.TLS, patch.TLS)
	return out
}

func mergeServerTLSConfig(current, patch ServerTLSConfig) ServerTLSConfig {
	out := current
	if patch.Enabled != current.Enabled {
		out.Enabled = patch.Enabled
	}
	if strings.TrimSpace(patch.CertFile) != "" {
		out.CertFile = patch.CertFile
	}
	if strings.TrimSpace(patch.KeyFile) != "" {
		out.KeyFile = patch.KeyFile
	}
	return out
}

func mergeDatabaseConfig(current, patch DatabaseConfig) DatabaseConfig {
	out := current
	if strings.TrimSpace(patch.Driver) != "" {
		out.Driver = patch.Driver
	}
	if strings.TrimSpace(patch.Path) != "" {
		out.Path = patch.Path
	}
	if strings.TrimSpace(patch.Host) != "" {
		out.Host = patch.Host
	}
	if patch.Port > 0 {
		out.Port = patch.Port
	}
	if strings.TrimSpace(patch.User) != "" {
		out.User = patch.User
	}
	if shouldApplySecret(patch.Password) {
		out.Password = patch.Password
	}
	if strings.TrimSpace(patch.PasswordFile) != "" {
		out.PasswordFile = patch.PasswordFile
	}
	if strings.TrimSpace(patch.DBName) != "" {
		out.DBName = patch.DBName
	}
	if strings.TrimSpace(patch.SSLMode) != "" {
		out.SSLMode = patch.SSLMode
	}
	if patch.MaxOpenConns > 0 {
		out.MaxOpenConns = patch.MaxOpenConns
	}
	if patch.MaxIdleConns > 0 {
		out.MaxIdleConns = patch.MaxIdleConns
	}
	return out
}

func mergeCAConfig(current, patch CAConfig) CAConfig {
	out := current
	if strings.TrimSpace(patch.StepCAURL) != "" {
		out.StepCAURL = patch.StepCAURL
	}
	if strings.TrimSpace(patch.RootPath) != "" {
		out.RootPath = patch.RootPath
	}
	if strings.TrimSpace(patch.IntermediatePath) != "" {
		out.IntermediatePath = patch.IntermediatePath
	}
	if strings.TrimSpace(patch.ProvisionerName) != "" {
		out.ProvisionerName = patch.ProvisionerName
	}
	if strings.TrimSpace(patch.MaxTTL) != "" {
		out.MaxTTL = patch.MaxTTL
	}
	if shouldApplySecret(patch.Password) {
		out.Password = patch.Password
	}
	if strings.TrimSpace(patch.PasswordFile) != "" {
		out.PasswordFile = patch.PasswordFile
	}
	if strings.TrimSpace(patch.ProvisionerPasswordFile) != "" {
		out.ProvisionerPasswordFile = patch.ProvisionerPasswordFile
	}
	out.Provisioners = mergeCAProvisionersConfig(current.Provisioners, patch.Provisioners)
	return out
}

func mergeCAProvisionersConfig(current, patch CAProvisionersConfig) CAProvisionersConfig {
	out := current
	if patch.ACME.Enabled != nil {
		out.ACME.Enabled = patch.ACME.Enabled
	}
	if patch.ACME.RequireEAB {
		out.ACME.RequireEAB = patch.ACME.RequireEAB
	}
	if len(patch.ACME.Challenges) > 0 {
		out.ACME.Challenges = append([]string(nil), patch.ACME.Challenges...)
	}
	if patch.ACME.DeviceAttestation {
		out.ACME.DeviceAttestation = patch.ACME.DeviceAttestation
	}
	if patch.SCEP.Enabled != nil {
		out.SCEP.Enabled = patch.SCEP.Enabled
	}
	if patch.SCEP.DeviceAttestation {
		out.SCEP.DeviceAttestation = patch.SCEP.DeviceAttestation
	}
	if shouldApplySecret(patch.SCEP.ChallengePassword) {
		out.SCEP.ChallengePassword = patch.SCEP.ChallengePassword
	}
	return out
}

func mergeSecurityConfig(current, patch SecurityConfig) SecurityConfig {
	out := current
	if shouldApplySecret(patch.JWTSecret) {
		out.JWTSecret = patch.JWTSecret
	}
	if patch.TokenExpirationHours > 0 {
		out.TokenExpirationHours = patch.TokenExpirationHours
	}
	if strings.TrimSpace(patch.CookieSameSite) != "" {
		out.CookieSameSite = patch.CookieSameSite
	}
	if patch.CookieSecure != nil {
		out.CookieSecure = patch.CookieSecure
	}
	out.Firewall = mergeFirewallConfig(current.Firewall, patch.Firewall)
	return out
}

func mergeFirewallConfig(current, patch FirewallConfig) FirewallConfig {
	out := current
	if patch.Enabled != current.Enabled {
		out.Enabled = patch.Enabled
	}
	if len(patch.Allowlist) > 0 {
		out.Allowlist = append([]string(nil), patch.Allowlist...)
	}
	return out
}

func mergeBootstrap(current, patch Bootstrap) Bootstrap {
	out := current
	if strings.TrimSpace(patch.AdminEmail) != "" {
		out.AdminEmail = patch.AdminEmail
	}
	if shouldApplySecret(patch.AdminPassword) {
		out.AdminPassword = patch.AdminPassword
	}
	return out
}

func mergeWebUIConfig(current, patch WebUIConfig) WebUIConfig {
	out := current
	out.Enabled = patch.Enabled
	if strings.TrimSpace(patch.UIDir) != "" {
		out.UIDir = patch.UIDir
	}
	if strings.TrimSpace(patch.PathPrefix) != "" {
		out.PathPrefix = patch.PathPrefix
	}
	if strings.TrimSpace(patch.ListenAddress) != "" {
		out.ListenAddress = patch.ListenAddress
	}
	if patch.ProxyAPI != nil {
		out.ProxyAPI = patch.ProxyAPI
	}
	if patch.MaxBodySize > 0 {
		out.MaxBodySize = patch.MaxBodySize
	}
	if strings.TrimSpace(patch.ReadTimeout) != "" {
		out.ReadTimeout = patch.ReadTimeout
	}
	if strings.TrimSpace(patch.WriteTimeout) != "" {
		out.WriteTimeout = patch.WriteTimeout
	}
	out.TLS = mergeWebUITLSConfig(current.TLS, patch.TLS)
	out.CORS = mergeWebUICORSConfig(current.CORS, patch.CORS)
	return out
}

func mergeWebUITLSConfig(current, patch WebUITLSConfig) WebUITLSConfig {
	out := current
	out.Enabled = patch.Enabled
	if strings.TrimSpace(patch.CertFile) != "" {
		out.CertFile = patch.CertFile
	}
	if strings.TrimSpace(patch.KeyFile) != "" {
		out.KeyFile = patch.KeyFile
	}
	return out
}

func mergeWebUICORSConfig(current, patch WebUICORSConfig) WebUICORSConfig {
	out := current
	if len(patch.AllowedOrigins) > 0 {
		out.AllowedOrigins = append([]string(nil), patch.AllowedOrigins...)
	}
	if len(patch.AllowedMethods) > 0 {
		out.AllowedMethods = append([]string(nil), patch.AllowedMethods...)
	}
	if len(patch.AllowedHeaders) > 0 {
		out.AllowedHeaders = append([]string(nil), patch.AllowedHeaders...)
	}
	out.AllowCredentials = patch.AllowCredentials
	return out
}

func shouldApplySecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == MaskedSecretValue {
		return false
	}
	return true
}

func validateServerConfig(cfg ServerConfig) error {
	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if cfg.Server.ReadTimeout < 0 || cfg.Server.WriteTimeout < 0 {
		return fmt.Errorf("server timeouts must not be negative")
	}
	if _, err := cfg.CA.MaxTTLDuration(); err != nil {
		return err
	}
	if _, err := cfg.Updater.CheckIntervalDuration(); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Updater.Channel) == "" {
		return fmt.Errorf("updater.channel is required")
	}
	if cfg.Database.MaxOpenConns < 0 || cfg.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database connection pool sizes must not be negative")
	}
	if cfg.Security.TokenExpirationHours < 0 {
		return fmt.Errorf("security.token_expiration_hours must not be negative")
	}
	for _, timeout := range []struct {
		name  string
		value string
	}{
		{"webui.read_timeout", cfg.WebUI.ReadTimeout},
		{"webui.write_timeout", cfg.WebUI.WriteTimeout},
	} {
		if strings.TrimSpace(timeout.value) == "" {
			continue
		}
		if _, err := time.ParseDuration(timeout.value); err != nil {
			return fmt.Errorf("parse %s %q: %w", timeout.name, timeout.value, err)
		}
	}
	return nil
}
