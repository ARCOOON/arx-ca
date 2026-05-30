package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TestServerConfigListenAddress(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfig
		want string
	}{
		{"host omitted", ServerConfig{Port: 8080}, ":8080"},
		{"explicit host", ServerConfig{Host: "127.0.0.1", Port: 9443}, "127.0.0.1:9443"},
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
	if got.Port != defaults.Port || got.CAConfigPath != defaults.CAConfigPath {
		t.Fatalf("unexpected defaults in file: %+v", got)
	}

	// Second call must not overwrite an existing file.
	if err := os.WriteFile(path, []byte("host: custom\nport: 1\n"), 0o644); err != nil {
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

	activeServerConfig = ServerConfig{
		Host:         "",
		Port:         9090,
		CAConfigPath: "/tmp/ca.json",
		DBType:       "badgerv2",
	}
	viper.Reset()
	viper.Set("port", 9090)
	viper.Set("ca_config_path", "/tmp/ca.json")

	ApplyServerRuntimeFromViper()

	if got := os.Getenv("CA_API_LISTEN_ADDR"); got != ":9090" {
		t.Fatalf("CA_API_LISTEN_ADDR = %q, want :9090", got)
	}
	if got := os.Getenv("CA_API_CA_CONFIG"); got != "/tmp/ca.json" {
		t.Fatalf("CA_API_CA_CONFIG = %q, want /tmp/ca.json", got)
	}
}
