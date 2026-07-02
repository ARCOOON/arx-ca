package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// ServerTLSConfig holds TLS settings for the main API HTTP server.
type ServerTLSConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" mapstructure:"key_file" yaml:"key_file"`
}

// ServerSettings holds HTTP server bind and timeout options.
type ServerSettings struct {
	Host         string          `json:"host" mapstructure:"host" yaml:"host"`
	Port         int             `json:"port" mapstructure:"port" yaml:"port"`
	LogLevel     string          `json:"log_level" mapstructure:"log_level" yaml:"log_level"`
	ReadTimeout  time.Duration   `json:"read_timeout" mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration   `json:"write_timeout" mapstructure:"write_timeout" yaml:"write_timeout"`
	TLS          ServerTLSConfig `json:"tls" mapstructure:"tls" yaml:"tls"`
}

// DatabaseConfig holds application user-store database connection settings.
type DatabaseConfig struct {
	Driver       string `json:"driver" mapstructure:"driver" yaml:"driver"`
	Path         string `json:"path" mapstructure:"path" yaml:"path"`
	Host         string `json:"host" mapstructure:"host" yaml:"host"`
	Port         int    `json:"port" mapstructure:"port" yaml:"port"`
	User         string `json:"user" mapstructure:"user" yaml:"user"`
	Password     string `json:"password" mapstructure:"password" yaml:"password"`
	PasswordFile string `json:"password_file" mapstructure:"password_file" yaml:"password_file"`
	DBName       string `json:"dbname" mapstructure:"dbname" yaml:"dbname"`
	SSLMode      string `json:"sslmode" mapstructure:"sslmode" yaml:"sslmode"`
	MaxOpenConns int    `json:"max_open_conns" mapstructure:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns" mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
}

// CAConfig holds step-ca integration paths and provisioner settings.
type CAConfig struct {
	StepCAURL               string               `json:"stepca_url" mapstructure:"stepca_url" yaml:"stepca_url"`
	RootPath                string               `json:"root_path" mapstructure:"root_path" yaml:"root_path"`
	IntermediatePath        string               `json:"intermediate_path" mapstructure:"intermediate_path" yaml:"intermediate_path"`
	ProvisionerName         string               `json:"provisioner_name" mapstructure:"provisioner_name" yaml:"provisioner_name"`
	MaxTTL                  string               `json:"max_ttl" mapstructure:"max_ttl" yaml:"max_ttl"`
	Password                string               `json:"password" mapstructure:"password" yaml:"password"`
	PasswordFile            string               `json:"password_file" mapstructure:"password_file" yaml:"password_file"`
	ProvisionerPasswordFile string               `json:"provisioner_password_file" mapstructure:"provisioner_password_file" yaml:"provisioner_password_file"`
	Provisioners            CAProvisionersConfig `json:"provisioners" mapstructure:"provisioners" yaml:"provisioners"`
}

// MaxTTLDuration parses the configured maximum X.509 certificate lifetime.
func (c CAConfig) MaxTTLDuration() (time.Duration, error) {
	raw := strings.TrimSpace(c.MaxTTL)
	if raw == "" {
		raw = DefaultServerConfig().CA.MaxTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse ca.max_ttl %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("ca.max_ttl must be positive: %s", raw)
	}
	return d, nil
}

// SecurityConfig holds authentication and token policy for the API.
type SecurityConfig struct {
	JWTSecret            string `json:"jwt_secret" mapstructure:"jwt_secret" yaml:"jwt_secret"`
	TokenExpirationHours int    `json:"token_expiration_hours" mapstructure:"token_expiration_hours" yaml:"token_expiration_hours"`
	CookieSameSite       string `json:"cookie_same_site" mapstructure:"cookie_same_site" yaml:"cookie_same_site"`
	CookieSecure         *bool  `json:"cookie_secure" mapstructure:"cookie_secure" yaml:"cookie_secure"`
}

// Bootstrap holds first-run admin credentials seeded when the users table is empty.
type Bootstrap struct {
	AdminEmail    string `json:"admin_email" mapstructure:"admin_email" yaml:"admin_email"`
	AdminPassword string `json:"admin_password" mapstructure:"admin_password" yaml:"admin_password"`
}

// ServiceConfig holds systemd self-install parameters for Infrastructure as Code.
type ServiceConfig struct {
	RunAsUser  string `json:"run_as_user" mapstructure:"run_as_user" yaml:"run_as_user"`
	InstallDir string `json:"install_dir" mapstructure:"install_dir" yaml:"install_dir"`
}

// TelemetryConfig holds OpenTelemetry exporter settings.
type TelemetryConfig struct {
	ServiceName      string `json:"service_name" mapstructure:"service_name" yaml:"service_name"`
	ExporterEndpoint string `json:"exporter_endpoint" mapstructure:"exporter_endpoint" yaml:"exporter_endpoint"`
	ExporterInsecure bool   `json:"exporter_insecure" mapstructure:"exporter_insecure" yaml:"exporter_insecure"`
	SDKDisabled      bool   `json:"sdk_disabled" mapstructure:"sdk_disabled" yaml:"sdk_disabled"`
}

// WebUITLSConfig holds TLS settings for the dedicated WebUI HTTP server.
type WebUITLSConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" mapstructure:"key_file" yaml:"key_file"`
}

// WebUICORSConfig holds CORS policy for the WebUI listener and API cross-origin access.
type WebUICORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins" mapstructure:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods" mapstructure:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers" mapstructure:"allowed_headers" yaml:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials" mapstructure:"allow_credentials" yaml:"allow_credentials"`
}

// UpdaterConfig controls the background release checker and optional auto-apply engine.
type UpdaterConfig struct {
	Enabled                  bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	Channel                  string `json:"channel" mapstructure:"channel" yaml:"channel"`
	NotifyOnly               bool   `json:"notify_only" mapstructure:"notify_only" yaml:"notify_only"`
	CheckInterval            string `json:"check_interval" mapstructure:"check_interval" yaml:"check_interval"`
	ViewChangelogAfterUpdate bool   `json:"view_changelog_after_update" mapstructure:"view_changelog_after_update" yaml:"view_changelog_after_update"`
}

// CheckIntervalDuration parses the configured background poll interval.
func (u UpdaterConfig) CheckIntervalDuration() (time.Duration, error) {
	raw := strings.TrimSpace(u.CheckInterval)
	if raw == "" {
		raw = DefaultServerConfig().Updater.CheckInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse updater.check_interval %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("updater.check_interval must be positive: %s", raw)
	}
	return d, nil
}

// NormalizedChannel returns the release track identifier used for GitHub polling.
func (u UpdaterConfig) NormalizedChannel() string {
	channel := strings.TrimSpace(strings.ToLower(u.Channel))
	if channel == "" {
		return "main"
	}
	return channel
}

// WebUIConfig holds the isolated WebUI static file server settings.
type WebUIConfig struct {
	Enabled       bool            `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	UIDir         string          `json:"ui_dir" mapstructure:"ui_dir" yaml:"ui_dir"`
	PathPrefix    string          `json:"path_prefix" mapstructure:"path_prefix" yaml:"path_prefix"`
	ListenAddress string          `json:"listen_address" mapstructure:"listen_address" yaml:"listen_address"`
	ProxyAPI      *bool           `json:"proxy_api" mapstructure:"proxy_api" yaml:"proxy_api"`
	MaxBodySize   int64           `json:"max_body_size" mapstructure:"max_body_size" yaml:"max_body_size"`
	ReadTimeout   string          `json:"read_timeout" mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout  string          `json:"write_timeout" mapstructure:"write_timeout" yaml:"write_timeout"`
	TLS           WebUITLSConfig  `json:"tls" mapstructure:"tls" yaml:"tls"`
	CORS          WebUICORSConfig `json:"cors" mapstructure:"cors" yaml:"cors"`
}

// ServerConfig is the root configuration loaded from server.yaml.
type ServerConfig struct {
	Server      ServerSettings    `mapstructure:"server" yaml:"server"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	CA          CAConfig          `mapstructure:"ca" yaml:"ca"`
	CABootstrap CABootstrapConfig `mapstructure:"ca_bootstrap" yaml:"ca_bootstrap"`
	Security    SecurityConfig    `mapstructure:"security" yaml:"security"`
	Bootstrap   Bootstrap         `mapstructure:"bootstrap" yaml:"bootstrap"`
	Telemetry   TelemetryConfig   `mapstructure:"telemetry" yaml:"telemetry"`
	Service     ServiceConfig     `mapstructure:"service" yaml:"service"`
	WebUI       WebUIConfig       `mapstructure:"webui" yaml:"webui"`
	Updater     UpdaterConfig     `mapstructure:"updater" yaml:"updater"`
}

// DefaultServerConfigForExecutable returns defaults with paths beside the current binary.
func DefaultServerConfigForExecutable() (ServerConfig, error) {
	cfg := DefaultServerConfig()
	exe, err := ExecutablePath()
	if err != nil {
		return ServerConfig{}, err
	}
	installDir := filepath.Dir(exe)
	cfg.WebUI.UIDir = filepath.Join(installDir, "ui")
	cfg.Service.InstallDir = installDir
	return cfg, nil
}

// webUIDirBesideExecutable returns the absolute path to the ui directory next to the executable.
func webUIDirBesideExecutable() (string, error) {
	exe, err := ExecutablePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "ui"), nil
}

// DefaultServerConfig returns the built-in defaults used when server.yaml is created.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Server: ServerSettings{
			Host:         "0.0.0.0",
			Port:         8080,
			LogLevel:     "info",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:       "sqlite",
			Path:         "arx.db",
			Host:         "127.0.0.1",
			Port:         5432,
			User:         "arx",
			DBName:       "arx_ca",
			SSLMode:      "disable",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		CA: CAConfig{
			RootPath:         ".pki/certs/root_ca.crt",
			IntermediatePath: ".pki/certs/intermediate_ca.crt",
			ProvisionerName:  "arx-admin",
			MaxTTL:           "87600h",
			Provisioners:     DefaultCAProvisionersConfig(),
		},
		Security: SecurityConfig{
			TokenExpirationHours: 24,
			CookieSameSite:       "lax",
		},
		CABootstrap: DefaultCABootstrapConfig(),
		Bootstrap: Bootstrap{
			AdminEmail:    "admin@arx.local",
			AdminPassword: "$2a$10$YGbMIqvYmKp3aQucKx0hh.x35Skzd9djQ/leMyXZy3JKKBVotzwxa",
		},
		Telemetry: TelemetryConfig{
			ServiceName:      "arx-ca",
			ExporterEndpoint: "http://localhost:4318",
			ExporterInsecure: false,
			SDKDisabled:      true,
		},
		Service: ServiceConfig{
			RunAsUser:  "arx-ca",
			InstallDir: "/opt/arx-ca",
		},
		Updater: UpdaterConfig{
			Enabled:                  true,
			Channel:                  "main",
			NotifyOnly:               true,
			CheckInterval:            "1h",
			ViewChangelogAfterUpdate: true,
		},
		WebUI: WebUIConfig{
			Enabled:       false,
			UIDir:         "/opt/arx-ca/ui",
			PathPrefix:    "/",
			ListenAddress: ":8443",
			MaxBodySize:   2 * 1024 * 1024,
			ReadTimeout:   "10s",
			WriteTimeout:  "10s",
			TLS: WebUITLSConfig{
				Enabled: true,
			},
			CORS: WebUICORSConfig{
				AllowedOrigins: []string{
					"http://localhost:5173",
					"http://127.0.0.1:5173",
				},
				AllowedMethods:   []string{"*"},
				AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept", "X-API-Key", "*"},
				AllowCredentials: true,
			},
		},
	}
}

// ListenAddress returns the HTTP listen address in host:port form.
func (c ServerConfig) ListenAddress() string {
	return c.Server.ListenAddress()
}

// ListenAddress returns the HTTP listen address for server settings.
func (s ServerSettings) ListenAddress() string {
	port := s.Port
	if port <= 0 {
		port = DefaultServerConfig().Server.Port
	}
	host := strings.TrimSpace(s.Host)
	if host == "" || host == "0.0.0.0" {
		return fmt.Sprintf(":%d", port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ResolvedTLSCertFile returns the absolute TLS certificate path for the API server.
func (s ServerSettings) ResolvedTLSCertFile() (string, error) {
	return resolveWebUIPath(s.TLS.CertFile)
}

// ResolvedTLSKeyFile returns the absolute TLS private key path for the API server.
func (s ServerSettings) ResolvedTLSKeyFile() (string, error) {
	return resolveWebUIPath(s.TLS.KeyFile)
}

// ConfigPath returns the absolute or relative path to step-ca ca.json.
func (c CAConfig) ConfigPath() string {
	if root := strings.TrimSpace(c.RootPath); root != "" {
		base := filepath.Dir(filepath.Dir(root))
		return filepath.Join(base, "config", "ca.json")
	}
	return ".pki/config/ca.json"
}

// EffectiveDriver returns the normalized database driver name (sqlite or postgres).
// When driver is unset, a non-empty host selects postgres for backward compatibility.
func (d DatabaseConfig) EffectiveDriver() string {
	driver := strings.ToLower(strings.TrimSpace(d.Driver))
	switch driver {
	case "postgres", "postgresql":
		return "postgres"
	case "sqlite", "sqlite3":
		return "sqlite"
	case "":
		if strings.TrimSpace(d.Host) != "" {
			return "postgres"
		}
		return strings.ToLower(strings.TrimSpace(DefaultServerConfig().Database.Driver))
	default:
		return driver
	}
}

// UsesPostgreSQL reports whether the application database should use PostgreSQL.
func (d DatabaseConfig) UsesPostgreSQL() bool {
	return d.EffectiveDriver() == "postgres"
}

// UsesSQLite reports whether the application database should use SQLite.
func (d DatabaseConfig) UsesSQLite() bool {
	return d.EffectiveDriver() == "sqlite"
}

// ResolvedSQLitePath returns the absolute path to the SQLite database file.
func (d DatabaseConfig) ResolvedSQLitePath() (string, error) {
	path := strings.TrimSpace(d.Path)
	if path == "" {
		path = DefaultServerConfig().Database.Path
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	baseDir, err := serverConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve sqlite database directory: %w", err)
	}
	return filepath.Join(baseDir, path), nil
}

// DSN builds a PostgreSQL connection string when the driver is postgres.
func (d DatabaseConfig) DSN() string {
	if !d.UsesPostgreSQL() {
		return ""
	}
	port := d.Port
	if port <= 0 {
		port = DefaultServerConfig().Database.Port
	}
	sslMode := strings.TrimSpace(d.SSLMode)
	if sslMode == "" {
		sslMode = DefaultServerConfig().Database.SSLMode
	}
	password := ResolveSecret(d.Password, d.PasswordFile)
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, password),
		Host:   fmt.Sprintf("%s:%d", strings.TrimSpace(d.Host), port),
		Path:   "/" + strings.TrimSpace(d.DBName),
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// TokenExpiration returns admin JWT lifetime derived from configuration.
func (s SecurityConfig) TokenExpiration() time.Duration {
	hours := s.TokenExpirationHours
	if hours <= 0 {
		hours = DefaultServerConfig().Security.TokenExpirationHours
	}
	return time.Duration(hours) * time.Hour
}

// ProxyAPIEnabled reports whether the WebUI listener should reverse-proxy API routes.
func (w WebUIConfig) ProxyAPIEnabled() bool {
	if w.ProxyAPI != nil {
		return *w.ProxyAPI
	}
	return true
}

// NormalizedPathPrefix returns the URL path prefix for WebUI routing (always starts with /).
func (w WebUIConfig) NormalizedPathPrefix() string {
	p := strings.TrimSpace(w.PathPrefix)
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(p, "/")
}

// EffectiveListenAddress returns the WebUI bind address.
func (w WebUIConfig) EffectiveListenAddress() string {
	addr := strings.TrimSpace(w.ListenAddress)
	if addr == "" {
		return DefaultServerConfig().WebUI.ListenAddress
	}
	return addr
}

// ReadTimeoutDuration parses the configured read timeout string.
func (w WebUIConfig) ReadTimeoutDuration() (time.Duration, error) {
	return parseWebUIDuration(w.ReadTimeout, DefaultServerConfig().WebUI.ReadTimeout)
}

// WriteTimeoutDuration parses the configured write timeout string.
func (w WebUIConfig) WriteTimeoutDuration() (time.Duration, error) {
	return parseWebUIDuration(w.WriteTimeout, DefaultServerConfig().WebUI.WriteTimeout)
}

// StartupURL returns a human-readable base URL for startup logging.
func (w WebUIConfig) StartupURL() string {
	scheme := "http"
	if w.TLS.Enabled {
		scheme = "https"
	}
	host := w.EffectiveListenAddress()
	if strings.HasPrefix(host, ":") {
		host = "0.0.0.0" + host
	} else {
		host = formatListenHostForDisplay(host)
	}
	prefix := w.NormalizedPathPrefix()
	if prefix == "/" {
		return fmt.Sprintf("%s://%s/", scheme, host)
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, prefix)
}

// ResolvedUIDir returns the absolute path to static UI assets.
func (w WebUIConfig) ResolvedUIDir() (string, error) {
	dir := strings.TrimSpace(w.UIDir)
	if dir == "" {
		dir = DefaultServerConfig().WebUI.UIDir
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	baseDir, err := serverConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve webui ui_dir directory: %w", err)
	}
	return filepath.Join(baseDir, dir), nil
}

// ResolvedTLSCertFile returns the absolute TLS certificate path when configured.
func (w WebUIConfig) ResolvedTLSCertFile() (string, error) {
	return resolveWebUIPath(w.TLS.CertFile)
}

// ResolvedTLSKeyFile returns the absolute TLS private key path when configured.
func (w WebUIConfig) ResolvedTLSKeyFile() (string, error) {
	return resolveWebUIPath(w.TLS.KeyFile)
}

func parseWebUIDuration(value, fallback string) (time.Duration, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		s = fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", s, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive: %s", s)
	}
	return d, nil
}

func resolveWebUIPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	baseDir, err := serverConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve webui path: %w", err)
	}
	return filepath.Join(baseDir, path), nil
}

func formatListenHostForDisplay(host string) string {
	if host == "0.0.0.0" {
		return host
	}
	if strings.HasPrefix(host, "[") {
		if idx := strings.LastIndex(host, "]:"); idx >= 0 {
			h := host[1:idx]
			if h == "" || h == "::" || h == "0.0.0.0" {
				return "0.0.0.0" + host[idx+1:]
			}
		}
		return host
	}
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		h := host[:idx]
		if h == "" || h == "0.0.0.0" {
			return "0.0.0.0" + host[idx:]
		}
	}
	return host
}

// GenerateJWTSecret creates a standard base64-encoded secret from n random bytes for HS256 signing.
func GenerateJWTSecret(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
