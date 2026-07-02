package service

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

var errMissingLocalAppData = errors.New("LOCALAPPDATA environment variable is not set")

// InstallScope selects user-level or system-level daemon registration.
type InstallScope int

const (
	// InstallScopeUser installs under the current user's home directory.
	InstallScopeUser InstallScope = iota
	// InstallScopeSystem installs system-wide (root/Administrator required).
	InstallScopeSystem
)

// IsSystem reports whether the install targets a system-wide service.
func (s InstallScope) IsSystem() bool {
	return s == InstallScopeSystem
}

func binaryFileName() string {
	if runtime.GOOS == "windows" {
		return "arx-ca.exe"
	}
	return "arx-ca"
}

func defaultSystemInstallDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramFiles"), "arx-ca")
	}
	return "/opt/arx-ca"
}

func defaultUserInstallDir() (string, error) {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", errMissingLocalAppData
		}
		return filepath.Join(localAppData, "arx-ca"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".arx-ca"), nil
}
