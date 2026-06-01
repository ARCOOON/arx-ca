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

// ServerSettings holds HTTP server bind and timeout options.
type ServerSettings struct {
	Host         string        `mapstructure:"host" yaml:"host"`
	Port         int           `mapstructure:"port" yaml:"port"`
	LogLevel     string        `mapstructure:"log_level" yaml:"log_level"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
}

// DatabaseConfig holds application user-store database connection settings.
type DatabaseConfig struct {
	Driver       string `mapstructure:"driver" yaml:"driver"`
	Path         string `mapstructure:"path" yaml:"path"`
	Host         string `mapstructure:"host" yaml:"host"`
	Port         int    `mapstructure:"port" yaml:"port"`
	User         string `mapstructure:"user" yaml:"user"`
	Password     string `mapstructure:"password" yaml:"password"`
	PasswordFile string `mapstructure:"password_file" yaml:"password_file"`
	DBName       string `mapstructure:"dbname" yaml:"dbname"`
	SSLMode      string `mapstructure:"sslmode" yaml:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
}

// CAConfig holds step-ca integration paths and provisioner settings.
type CAConfig struct {
	StepCAURL               string `mapstructure:"stepca_url" yaml:"stepca_url"`
	RootPath                string `mapstructure:"root_path" yaml:"root_path"`
	IntermediatePath        string `mapstructure:"intermediate_path" yaml:"intermediate_path"`
	ProvisionerName         string `mapstructure:"provisioner_name" yaml:"provisioner_name"`
	Password                string `mapstructure:"password" yaml:"password"`
	PasswordFile            string `mapstructure:"password_file" yaml:"password_file"`
	ProvisionerPasswordFile string `mapstructure:"provisioner_password_file" yaml:"provisioner_password_file"`
}

// SecurityConfig holds authentication and token policy for the API.
type SecurityConfig struct {
	JWTSecret            string `mapstructure:"jwt_secret" yaml:"jwt_secret"`
	TokenExpirationHours int    `mapstructure:"token_expiration_hours" yaml:"token_expiration_hours"`
}

// Bootstrap holds first-run admin credentials seeded when the users table is empty.
type Bootstrap struct {
	AdminEmail        string `mapstructure:"admin_email" yaml:"admin_email"`
	AdminPasswordHash string `mapstructure:"admin_password_hash" yaml:"admin_password_hash"`
}

// TelemetryConfig holds OpenTelemetry exporter settings.
type TelemetryConfig struct {
	ServiceName      string `mapstructure:"service_name" yaml:"service_name"`
	ExporterEndpoint string `mapstructure:"exporter_endpoint" yaml:"exporter_endpoint"`
	ExporterInsecure bool   `mapstructure:"exporter_insecure" yaml:"exporter_insecure"`
	SDKDisabled      bool   `mapstructure:"sdk_disabled" yaml:"sdk_disabled"`
}

// ServerConfig is the root configuration loaded from server.yaml.
type ServerConfig struct {
	Server    ServerSettings  `mapstructure:"server" yaml:"server"`
	Database  DatabaseConfig  `mapstructure:"database" yaml:"database"`
	CA        CAConfig        `mapstructure:"ca" yaml:"ca"`
	Security  SecurityConfig  `mapstructure:"security" yaml:"security"`
	Bootstrap Bootstrap       `mapstructure:"bootstrap" yaml:"bootstrap"`
	Telemetry TelemetryConfig `mapstructure:"telemetry" yaml:"telemetry"`
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
			ProvisionerName:  "ca-admin",
		},
		Security: SecurityConfig{
			TokenExpirationHours: 24,
		},
		Bootstrap: Bootstrap{
			AdminEmail:        "admin@arx.local",
			AdminPasswordHash: "$2a$10$YGbMIqvYmKp3aQucKx0hh.x35Skzd9djQ/leMyXZy3JKKBVotzwxa",
		},
		Telemetry: TelemetryConfig{
			ServiceName:      "arx-ca",
			ExporterEndpoint: "http://localhost:4318",
			ExporterInsecure: true,
			SDKDisabled:      false,
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

// GenerateJWTSecret creates a URL-safe base64 secret for HS256 signing.
func GenerateJWTSecret(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
