package config

import "path/filepath"

// EnsureAgentConfigFile creates agent.yaml at path with built-in defaults when the file does not exist.
func EnsureAgentConfigFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	defaults := DefaultAgentConfig()
	return ensureYAMLConfigFile(abs, defaults, 0o600)
}
