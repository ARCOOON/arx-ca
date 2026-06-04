//go:build !linux && !windows

package service

import "fmt"

// Install registers the arx CA server daemon (Linux and Windows only).
func Install(_ InstallOptions) error {
	return fmt.Errorf("service install is only supported on Linux and Windows")
}

// Uninstall removes the arx CA server daemon (Linux and Windows only).
func Uninstall(_ InstallOptions) error {
	return fmt.Errorf("service uninstall is only supported on Linux and Windows")
}
