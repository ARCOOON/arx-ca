package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAgentConfig writes a template agent.yaml to path. When force is false and the file
// already exists, WriteAgentConfig returns an error.
func WriteAgentConfig(path string, force bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve agent config path: %w", err)
	}
	path = abs

	if _, err := os.Stat(path); err == nil {
		if !force {
			return fmt.Errorf("configuration already exists at %s. Use --force to overwrite", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	dirPerm := os.FileMode(0o755)
	if filepath.Base(dir) == agentConfigDirName {
		dirPerm = 0o700
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	defaults := TemplateAgentConfig()
	raw, err := marshalYAMLConfig(defaults)
	if err != nil {
		return fmt.Errorf("marshal agent config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}
