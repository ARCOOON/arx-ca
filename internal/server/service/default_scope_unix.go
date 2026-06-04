//go:build unix

package service

import "os"

// DefaultInstallScope selects system scope when the process is root, otherwise user scope.
func DefaultInstallScope() InstallScope {
	if os.Geteuid() == 0 {
		return InstallScopeSystem
	}
	return InstallScopeUser
}
