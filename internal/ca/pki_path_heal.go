package ca

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// healPKIConfigPaths rewrites ca.json filesystem paths when artifacts exist under
// basePath but recorded paths point at another environment (for example WSL /mnt/c/...).
func healPKIConfigPaths(configPath, basePath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read CA config for path heal: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse CA config for path heal: %w", err)
	}

	changed := false

	for _, key := range []string{"root", "crt", "key"} {
		current, _ := doc[key].(string)
		if healed, ok := healArtifactPath(current, artifactPathForRole(basePath, key)); ok {
			doc[key] = healed
			changed = true
		}
	}

	if ssh, ok := doc["ssh"].(map[string]any); ok {
		for jsonKey, rel := range map[string]string{
			"hostKey": filepath.Join("secrets", "ssh_host_ca_key"),
			"userKey": filepath.Join("secrets", "ssh_user_ca_key"),
		} {
			current, _ := ssh[jsonKey].(string)
			if healed, ok := healArtifactPath(current, filepath.Join(basePath, rel)); ok {
				ssh[jsonKey] = healed
				changed = true
			}
		}
	}

	if dbCfg, ok := doc["db"].(map[string]any); ok {
		current, _ := dbCfg["dataSource"].(string)
		dbType, _ := dbCfg["type"].(string)
		if isBadgerDBType(dbType) {
			if healed, ok := healArtifactPath(current, filepath.Join(basePath, "db")); ok {
				dbCfg["dataSource"] = healed
				changed = true
			}
		}
	}

	if !changed {
		return nil
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal healed CA config: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return fmt.Errorf("write healed CA config: %w", err)
	}
	log.Printf("ca: healed PKI filesystem paths in %s for the current platform", configPath)
	return nil
}

func isBadgerDBType(dbType string) bool {
	switch strings.ToLower(strings.TrimSpace(dbType)) {
	case "badger", "badgerv1", "badgerv2", "":
		return true
	default:
		return false
	}
}

func artifactPathForRole(basePath, role string) string {
	switch role {
	case "root":
		return filepath.Join(basePath, "certs", "root_ca.crt")
	case "crt":
		return filepath.Join(basePath, "certs", "intermediate_ca.crt")
	case "key":
		return filepath.Join(basePath, "secrets", "intermediate_ca_key")
	default:
		return ""
	}
}

func healArtifactPath(current, candidate string) (string, bool) {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return current, false
	}
	if pathExists(current) {
		return current, false
	}
	if !pathExists(candidate) {
		return current, false
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return candidate, true
	}
	return abs, true
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
