package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirName             = ".arx-cert-service"
	rootPEMName         = "root.pem"
	intermediatePEMName = "intermediate.pem"
	configName          = "config.json"
)

// Config records agent-local metadata for trust management.
type Config struct {
	APIURL string `json:"api_url,omitempty"`
}

// Dir returns the per-user agent state directory (~/.arx-cert-service).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, dirName), nil
}

func pemPath(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// RootPEMPath returns the path where the installed root CA PEM is stored.
func RootPEMPath() (string, error) {
	return pemPath(rootPEMName)
}

// IntermediatePEMPath returns the path where the installed intermediate CA PEM is stored.
func IntermediatePEMPath() (string, error) {
	return pemPath(intermediatePEMName)
}

// ConfigPath returns the path to the agent config file.
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// EnsureDir creates the agent state directory with restrictive permissions.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create state directory: %w", err)
	}
	return dir, nil
}

func savePEM(path, pem string) error {
	if _, err := EnsureDir(); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}
	return nil
}

func loadPEM(path, label string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no installed %s found at %s", label, path)
		}
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("stored %s is empty", label)
	}
	return data, nil
}

func removePEM(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove certificate: %w", err)
	}
	return nil
}

// SaveRootPEM writes the root certificate PEM to the state directory.
func SaveRootPEM(pem string) error {
	path, err := RootPEMPath()
	if err != nil {
		return err
	}
	return savePEM(path, pem)
}

// LoadRootPEM reads the stored root certificate PEM.
func LoadRootPEM() ([]byte, error) {
	path, err := RootPEMPath()
	if err != nil {
		return nil, err
	}
	return loadPEM(path, "root CA")
}

// RemoveRootPEM deletes the stored root certificate PEM if present.
func RemoveRootPEM() error {
	path, err := RootPEMPath()
	if err != nil {
		return err
	}
	return removePEM(path)
}

// SaveIntermediatePEM writes the intermediate certificate PEM to the state directory.
func SaveIntermediatePEM(pem string) error {
	path, err := IntermediatePEMPath()
	if err != nil {
		return err
	}
	return savePEM(path, pem)
}

// LoadIntermediatePEM reads the stored intermediate certificate PEM.
func LoadIntermediatePEM() ([]byte, error) {
	path, err := IntermediatePEMPath()
	if err != nil {
		return nil, err
	}
	return loadPEM(path, "intermediate CA")
}

// RemoveIntermediatePEM deletes the stored intermediate certificate PEM if present.
func RemoveIntermediatePEM() error {
	path, err := IntermediatePEMPath()
	if err != nil {
		return err
	}
	return removePEM(path)
}

// SaveConfig writes agent configuration.
func SaveConfig(cfg Config) error {
	if _, err := EnsureDir(); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// LoadConfig reads agent configuration (empty config if missing).
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}
