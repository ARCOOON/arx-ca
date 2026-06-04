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
		return "arx.exe"
	}
	return "arx"
}

func defaultSystemInstallDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramFiles"), "arx")
	}
	return "/opt/arx"
}

func defaultUserInstallDir() (string, error) {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", errMissingLocalAppData
		}
		return filepath.Join(localAppData, "arx"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".arx"), nil
}
