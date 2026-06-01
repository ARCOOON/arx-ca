//go:build !linux

package service

import "fmt"

// Install registers the arx-ca-server systemd unit (Linux only).
func Install(_ string) error {
	return fmt.Errorf("service install is only supported on Linux")
}

// Uninstall removes the arx-ca-server systemd unit (Linux only).
func Uninstall() error {
	return fmt.Errorf("service uninstall is only supported on Linux")
}
