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

// SetCLIServerURL updates server_url in ~/.arx-ca/cli.yaml and refreshes the active Viper state.
func SetCLIServerURL(url string) error {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return fmt.Errorf("server url is empty")
	}

	path, err := cliConfigFilePath()
	if err != nil {
		return err
	}

	cfg := CLIConfig{LogLevel: DefaultCLIConfig().LogLevel}
	if raw, err := os.ReadFile(path); err == nil {
		var data map[string]any
		if err := yaml.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("parse CLI config %s: %w", path, err)
		}
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "mapstructure",
			Result:  &cfg,
		})
		if err != nil {
			return fmt.Errorf("decode CLI config %s: %w", path, err)
		}
		if err := dec.Decode(data); err != nil {
			return fmt.Errorf("decode CLI config %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read CLI config %s: %w", path, err)
	}

	cfg.ServerURL = url
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultCLIConfig().LogLevel
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	out, err := marshalYAMLConfig(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write CLI config %s: %w", path, err)
	}

	viper.Set("server_url", url)
	activeCLIConfig.ServerURL = url
	return nil
}
