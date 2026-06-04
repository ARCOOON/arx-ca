package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// PersistServerConfig normalizes cfg and writes it to path with mode 0600.
func PersistServerConfig(path string, cfg ServerConfig) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve server config path: %w", err)
	}
	path = abs

	cfg = normalizeServerConfig(cfg)
	raw, err := marshalYAMLConfig(cfg)
	if err != nil {
		return fmt.Errorf("marshal server config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write server config %s: %w", path, err)
	}
	return nil
}
