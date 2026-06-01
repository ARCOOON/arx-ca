//go:build !linux

package service

import "fmt"

// Install registers the arx-server systemd unit (Linux only).
func Install(_ InstallOptions) error {
	return fmt.Errorf("service install is only supported on Linux")
}

// Uninstall removes the arx-server systemd unit (Linux only).
func Uninstall(_ InstallOptions) error {
	return fmt.Errorf("service uninstall is only supported on Linux")
}
