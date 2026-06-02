//go:build linux

package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	arxconfig "github.com/your-org/arx-ca/internal/config"
)

// Install copies the current binary to installDir, bootstraps agent.yaml, registers
// the arx-agent systemd unit, and starts the service. It must run as root.
func Install(opts InstallOptions) error {
	if err := requireRoot("install"); err != nil {
		return err
	}
	if err := requireSystemctl(); err != nil {
		return err
	}

	user := opts.runAsUser()
	installDir := opts.installDir()
	destBinary := filepath.Join(installDir, binaryName)
	configPath := filepath.Join(installDir, configFileName)

	if err := ensureSystemUser(user); err != nil {
		return err
	}

	if err := os.MkdirAll(installDir, 0o700); err != nil {
		return fmt.Errorf("create install directory %s: %w", installDir, err)
	}

	srcBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	if err := copyBinary(srcBinary, destBinary); err != nil {
		return err
	}

	if err := bootstrapAgentConfig(configPath); err != nil {
		return err
	}

	if err := setInstallPermissions(user, installDir, destBinary, configPath); err != nil {
		return err
	}

	params := UnitParams{
		RunAsUser:  user,
		InstallDir: installDir,
		ExecPath:   destBinary,
		ConfigPath: configPath,
	}
	if err := writeUnitFile(params); err != nil {
		return err
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl("enable", unitName); err != nil {
		return err
	}
	if err := runSystemctl("restart", unitName); err != nil {
		return err
	}

	fmt.Println("arx-agent installed and started.")
	fmt.Printf("Service:  %s\n", unitName)
	fmt.Printf("Binary:   %s\n", destBinary)
	fmt.Printf("Config:   %s\n", configPath)
	fmt.Println("Edit agent.yaml (managed_certs, thresholds) and ensure admin credentials are available for renewal.")
	return nil
}

// Uninstall stops the arx-agent unit, removes the unit file and install directory, and
// deletes the service user. It must run as root.
func Uninstall(opts InstallOptions) error {
	if err := requireRoot("uninstall"); err != nil {
		return err
	}
	if err := requireSystemctl(); err != nil {
		return err
	}

	user := opts.runAsUser()
	installDir := opts.installDir()

	_ = runSystemctl("stop", unitName)
	_ = runSystemctl("disable", unitName)

	if err := os.Remove(unitFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", unitFilePath, err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}

	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove install directory %s: %w", installDir, err)
	}

	if err := removeSystemUser(user); err != nil {
		return err
	}

	fmt.Println("arx-agent uninstalled.")
	return nil
}

func bootstrapAgentConfig(configPath string) error {
	return arxconfig.EnsureAgentConfigFile(configPath)
}

func requireRoot(action string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service %s must be executed as root", action)
	}
	return nil
}

func requireSystemctl() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errors.New("systemd (systemctl) is required but was not found")
	}
	return nil
}

func ensureSystemUser(user string) error {
	if exec.Command("id", user).Run() == nil {
		return nil
	}
	cmd := exec.Command(
		"useradd",
		"--system",
		"--no-create-home",
		"--shell", "/usr/sbin/nologin",
		user,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if exec.Command("id", user).Run() == nil {
			return nil
		}
		return fmt.Errorf("create system user %s: %w: %s", user, err, trimOutput(out))
	}
	return nil
}

func removeSystemUser(user string) error {
	if exec.Command("id", user).Run() != nil {
		return nil
	}
	cmd := exec.Command("userdel", user)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove system user %s: %w: %s", user, err, trimOutput(out))
	}
	return nil
}

func copyBinary(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source binary %s: %w", src, err)
	}
	defer in.Close()

	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing binary %s: %w", dst, err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("create destination binary %s: %w", dst, err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy binary to %s: %w", dst, err)
	}
	if err := out.Chmod(0o700); err != nil {
		return fmt.Errorf("chmod destination binary %s: %w", dst, err)
	}
	return nil
}

func setInstallPermissions(user, installDir, destBinary, configPath string) error {
	chown := exec.Command("chown", "-R", user+":"+user, installDir)
	if out, err := chown.CombinedOutput(); err != nil {
		return fmt.Errorf("chown %s: %w: %s", installDir, err, trimOutput(out))
	}

	if err := os.Chmod(installDir, 0o700); err != nil {
		return fmt.Errorf("chmod install directory: %w", err)
	}
	if err := os.Chmod(destBinary, 0o700); err != nil {
		return fmt.Errorf("chmod installed binary: %w", err)
	}
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Chmod(configPath, 0o600); err != nil {
			return fmt.Errorf("chmod agent config: %w", err)
		}
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
