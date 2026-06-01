//go:build linux

package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	arxconfig "github.com/your-org/arx-ca/internal/config"
)

// Install registers and starts the arx-ca-server systemd unit. It must run as root.
func Install(configFlag string) error {
	if os.Geteuid() != 0 {
		return errors.New("service install must be executed as root")
	}

	execPath, err := arxconfig.ExecutablePath()
	if err != nil {
		return err
	}

	configPath, err := arxconfig.ResolveServerConfigPath(configFlag)
	if err != nil {
		return err
	}

	workingDir := filepath.Dir(execPath)

	if err := ensureSystemUser(); err != nil {
		return err
	}

	params := UnitParams{
		ExecPath:   execPath,
		ConfigPath: configPath,
		WorkingDir: workingDir,
	}
	if err := writeUnitFile(params); err != nil {
		return err
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("enable", "--now", unitName); err != nil {
		return err
	}

	fmt.Println("arx-ca-server systemd service installed and started.")
	fmt.Println("Ensure the arx-ca user can read the executable, configuration file, and any paths referenced in server.yaml (CA keys, database secrets, etc.).")
	return nil
}

// Uninstall stops and removes the arx-ca-server systemd unit. It must run as root.
func Uninstall() error {
	if os.Geteuid() != 0 {
		return errors.New("service uninstall must be executed as root")
	}

	_ = runSystemctl("stop", unitName)
	if err := runSystemctl("disable", unitName); err != nil {
		return err
	}

	if err := os.Remove(unitFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", unitFilePath, err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}

	fmt.Println("arx-ca-server systemd service uninstalled.")
	return nil
}

func ensureSystemUser() error {
	if exec.Command("id", "-u", systemUser).Run() == nil {
		return nil
	}
	cmd := exec.Command("useradd", "--system", "-M", "-s", "/usr/sbin/nologin", systemUser)
	if err := cmd.Run(); err != nil {
		if exec.Command("id", "-u", systemUser).Run() == nil {
			return nil
		}
		return fmt.Errorf("create system user %s: %w", systemUser, err)
	}
	return nil
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w: %s", args[0], err, trimOutput(out))
	}
	return nil
}

func trimOutput(b []byte) string {
	const maxLen = 512
	s := string(b)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
