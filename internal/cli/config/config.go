package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const configDirName = ".arx"
const configFileName = "config.json"

// Config holds persisted CLI authentication and server settings.
type Config struct {
	ServerURL string    `json:"server_url"`
	Token     string    `json:"token"`
	TokenType string    `json:"token_type,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Username  string    `json:"username,omitempty"`
}

// Path returns the default config file path (~/.arx/config.json).
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

// Load reads the CLI config from disk. Missing file returns a zero Config and nil error.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes cfg to ~/.arx/config.json with restrictive permissions.
func Save(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	path, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// BearerToken returns the value for the Authorization header.
func (c *Config) BearerToken() string {
	if c == nil || c.Token == "" {
		return ""
	}
	tokenType := c.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return tokenType + " " + c.Token
}
