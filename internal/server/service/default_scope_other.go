//go:build !unix && !windows

package service

// DefaultInstallScope returns user scope on unsupported platforms.
func DefaultInstallScope() InstallScope {
	return InstallScopeUser
}
