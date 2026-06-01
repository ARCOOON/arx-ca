package config

import (
	"os"
	"strings"
)

// ResolveSecret returns the trimmed contents of filePath when set, otherwise value.
func ResolveSecret(value, filePath string) string {
	path := strings.TrimSpace(filePath)
	if path == "" {
		return value
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return value
	}
	return strings.TrimSpace(string(raw))
}
