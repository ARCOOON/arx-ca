package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ReadServerConfig loads server.toml from the path resolved by configFlag.
// When the file does not exist, it returns found=false and a zero ServerConfig without error.
func ReadServerConfig(configFlag string) (ServerConfig, bool, error) {
	path, err := ResolveServerConfigPath(configFlag)
	if err != nil {
		return ServerConfig{}, false, fmt.Errorf("resolve server config path: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ServerConfig{}, false, nil
		}
		return ServerConfig{}, false, fmt.Errorf("read server config %s: %w", path, err)
	}

	var cfg ServerConfig
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return ServerConfig{}, false, fmt.Errorf("parse server config %s: %w", path, err)
	}
	return cfg, true, nil
}

// ServiceInstallSettingsFromConfig returns service install fields from server.toml when present.
func ServiceInstallSettingsFromConfig(configFlag string) (runAsUser, installDir string, err error) {
	cfg, found, err := ReadServerConfig(configFlag)
	if err != nil {
		return "", "", err
	}
	if !found {
		return "", "", nil
	}
	return strings.TrimSpace(cfg.Service.RunAsUser), strings.TrimSpace(cfg.Service.InstallDir), nil
}
