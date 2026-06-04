//go:build windows

package service

// DefaultInstallScope selects system scope when the process is elevated, otherwise user scope.
func DefaultInstallScope() InstallScope {
	if isWindowsAdministrator() {
		return InstallScopeSystem
	}
	return InstallScopeUser
}
