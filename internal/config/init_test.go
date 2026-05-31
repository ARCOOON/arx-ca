package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TestServerConfigListenAddress(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfig
		want string
	}{
		{
			name: "default host",
			cfg: ServerConfig{Server: ServerSettings{Host: "0.0.0.0", Port: 8080}},
			want: ":8080",
		},
		{
			name: "explicit host",
			cfg:  ServerConfig{Server: ServerSettings{Host: "127.0.0.1", Port: 9443}},
			want: "127.0.0.1:9443",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.ListenAddress(); got != tc.want {
				t.Fatalf("ListenAddress() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnsureYAMLConfigFileCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, serverConfigFileName)
	defaults := DefaultServerConfig()

	if err := ensureYAMLConfigFile(path, defaults, 0o644); err != nil {
		t.Fatalf("ensureYAMLConfigFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created config: %v", err)
	}

	var got ServerConfig
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal created config: %v", err)
	}
	if got.Server.Port != defaults.Server.Port {
		t.Fatalf("server.port = %d, want %d", got.Server.Port, defaults.Server.Port)
	}
	if got.CA.ConfigPath() != defaults.CA.ConfigPath() {
		t.Fatalf("ca config path = %q, want %q", got.CA.ConfigPath(), defaults.CA.ConfigPath())
	}
	if !strings.Contains(string(raw), "server:") || !strings.Contains(string(raw), "database:") {
		t.Fatalf("expected nested yaml sections, got:\n%s", string(raw))
	}
	if !strings.Contains(string(raw), "password_file:") {
		t.Fatalf("expected password_file fields in generated config, got:\n%s", string(raw))
	}
	if !strings.Contains(string(raw), "admin_password_hash:") {
		t.Fatalf("expected admin_password_hash in generated config, got:\n%s", string(raw))
	}

	if err := os.WriteFile(path, []byte("server:\n  host: custom\n  port: 1\n"), 0o644); err != nil {
		t.Fatalf("overwrite config for idempotency test: %v", err)
	}
	if err := ensureYAMLConfigFile(path, defaults, 0o644); err != nil {
		t.Fatalf("ensureYAMLConfigFile second call: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after second call: %v", err)
	}
	if !strings.Contains(string(after), "custom") {
		t.Fatalf("existing config was overwritten")
	}
}

func TestInitCLIConfigUsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	viper.Reset()
	if err := InitCLIConfig(); err != nil {
		t.Fatalf("InitCLIConfig: %v", err)
	}

	path := filepath.Join(home, cliConfigDirName, cliConfigFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config at %s: %v", path, err)
	}

	cfg := CLIConfigFromViper()
	if cfg.ServerURL != DefaultCLIConfig().ServerURL {
		t.Fatalf("server_url = %q, want %q", cfg.ServerURL, DefaultCLIConfig().ServerURL)
	}
	if viper.GetString("server_url") != cfg.ServerURL {
		t.Fatalf("viper server_url = %q, want %q", viper.GetString("server_url"), cfg.ServerURL)
	}
}

func TestApplyServerRuntimeFromViperSetsEnv(t *testing.T) {
	t.Setenv("CA_API_LISTEN_ADDR", "")
	t.Setenv("CA_API_CA_CONFIG", "")
	t.Setenv("CA_API_JWT_SECRET", "")

	activeServerConfig = ServerConfig{
		Server: ServerSettings{Host: "0.0.0.0", Port: 9090},
		CA:     CAConfig{RootPath: "/tmp/pki/certs/root_ca.crt"},
		Security: SecurityConfig{
			JWTSecret:            "test-secret",
			TokenExpirationHours: 12,
		},
	}
	viper.Reset()
	viper.Set("server.port", 9090)
	viper.Set("ca.root_path", "/tmp/pki/certs/root_ca.crt")
	viper.Set("security.jwt_secret", "test-secret")

	ApplyServerRuntimeFromViper()

	if got := os.Getenv("CA_API_LISTEN_ADDR"); got != ":9090" {
		t.Fatalf("CA_API_LISTEN_ADDR = %q, want :9090", got)
	}
	wantCA := filepath.Join("/tmp", "pki", "config", "ca.json")
	if got := os.Getenv("CA_API_CA_CONFIG"); got != wantCA {
		t.Fatalf("CA_API_CA_CONFIG = %q, want %q", got, wantCA)
	}
	if got := os.Getenv("CA_API_JWT_SECRET"); got != "test-secret" {
		t.Fatalf("CA_API_JWT_SECRET = %q, want test-secret", got)
	}
}

func TestNormalizeServerConfigGeneratesJWTSecret(t *testing.T) {
	t.Setenv("CA_API_JWT_SECRET", "")
	t.Setenv("ARX_SECURITY_JWT_SECRET", "")

	cfg := normalizeServerConfig(ServerConfig{
		Security: SecurityConfig{TokenExpirationHours: 8},
	})
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		t.Fatal("expected generated JWT secret")
	}
	if cfg.Security.TokenExpiration() != 8*time.Hour {
		t.Fatalf("token expiration = %v, want 8h", cfg.Security.TokenExpiration())
	}
}

func TestDatabaseDSN(t *testing.T) {
	dsn := DatabaseConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "arx",
		Password: "secret",
		DBName:   "arx_ca",
		SSLMode:  "require",
	}.DSN()
	if !strings.Contains(dsn, "db.example.com:5432") || !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("unexpected DSN: %s", dsn)
	}
}
