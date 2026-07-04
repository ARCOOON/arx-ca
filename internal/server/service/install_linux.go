//go:build linux

package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Install registers the arx systemd unit for the selected scope and starts the service.
func Install(opts InstallOptions) error {
	if opts.Scope.IsSystem() {
		return installSystem(opts)
	}
	return installUser(opts)
}

// Uninstall stops and removes the arx systemd unit and install directory for the selected scope.
func Uninstall(opts InstallOptions) error {
	if opts.Scope.IsSystem() {
		return uninstallSystem(opts)
	}
	return uninstallUser(opts)
}

func installSystem(opts InstallOptions) error {
	if err := requireRoot("install"); err != nil {
		return err
	}
	if err := requireSystemctl(); err != nil {
		return err
	}

	user := opts.runAsUser()
	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}
	destBinary := filepath.Join(installDir, binaryFileName())
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
	if err := bootstrapConfig(installDir, destBinary); err != nil {
		return err
	}
	if err := setSystemInstallPermissions(user, installDir, destBinary, configPath); err != nil {
		return err
	}

	target, err := unitTargetForScope(InstallScopeSystem)
	if err != nil {
		return err
	}
	params := UnitParams{
		RunAsUser:  user,
		InstallDir: installDir,
		ExecPath:   destBinary,
		ConfigPath: configPath,
	}
	if err := writeUnitFile(target, params); err != nil {
		return err
	}
	_ = os.Remove(legacySystemUnitPath())

	if err := runSystemctl(false, "daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl(false, "enable", unitName); err != nil {
		return err
	}
	if err := runSystemctl(false, "restart", unitName); err != nil {
		return err
	}

	printInstallSuccess("system", unitName, destBinary, configPath, target.filePath)
	return nil
}

func installUser(opts InstallOptions) error {
	if err := requireSystemctl(); err != nil {
		return err
	}

	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}
	destBinary := filepath.Join(installDir, binaryFileName())
	configPath := filepath.Join(installDir, configFileName)

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
	if err := bootstrapConfig(installDir, destBinary); err != nil {
		return err
	}
	if err := os.Chmod(configPath, 0o600); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("chmod server config: %w", err)
	}

	target, err := unitTargetForScope(InstallScopeUser)
	if err != nil {
		return err
	}
	params := UnitParams{
		InstallDir: installDir,
		ExecPath:   destBinary,
		ConfigPath: configPath,
	}
	if err := writeUnitFile(target, params); err != nil {
		return err
	}

	_ = enableUserLinger()

	if err := runSystemctl(true, "daemon-reload"); err != nil {
		return err
	}
	if err := runSystemctl(true, "enable", unitName); err != nil {
		return err
	}
	if err := runSystemctl(true, "restart", unitName); err != nil {
		return err
	}

	printInstallSuccess("user", unitName, destBinary, configPath, target.filePath)
	fmt.Println("User services run under systemd --user. Ensure lingering is enabled for headless operation:")
	fmt.Println("  loginctl enable-linger $USER")
	return nil
}

func uninstallSystem(opts InstallOptions) error {
	if err := requireRoot("uninstall"); err != nil {
		return err
	}
	if err := requireSystemctl(); err != nil {
		return err
	}

	user := opts.runAsUser()
	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}

	target, err := unitTargetForScope(InstallScopeSystem)
	if err != nil {
		return err
	}

	_ = runSystemctl(false, "stop", unitName)
	_ = runSystemctl(false, "disable", unitName)
	_ = os.Remove(target.filePath)
	_ = os.Remove(legacySystemUnitPath())

	if err := runSystemctl(false, "daemon-reload"); err != nil {
		return err
	}
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove install directory %s: %w", installDir, err)
	}
	if err := removeSystemUser(user); err != nil {
		return err
	}

	fmt.Println("arx-ca CA server uninstalled (system scope).")
	return nil
}

func uninstallUser(opts InstallOptions) error {
	if err := requireSystemctl(); err != nil {
		return err
	}

	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}

	target, err := unitTargetForScope(InstallScopeUser)
	if err != nil {
		return err
	}

	_ = runSystemctl(true, "stop", unitName)
	_ = runSystemctl(true, "disable", unitName)
	_ = os.Remove(target.filePath)

	if err := runSystemctl(true, "daemon-reload"); err != nil {
		return err
	}
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove install directory %s: %w", installDir, err)
	}

	fmt.Println("arx-ca CA server uninstalled (user scope).")
	return nil
}

func requireRoot(action string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service %s with --system requires root privileges", action)
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

func setSystemInstallPermissions(user, installDir, destBinary, configPath string) error {
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
			return fmt.Errorf("chmod server config: %w", err)
		}
	}
	return nil
}

func runSystemctl(userMode bool, args ...string) error {
	cmdArgs := append([]string{}, args...)
	if userMode {
		cmdArgs = append([]string{"--user"}, cmdArgs...)
	}
	cmd := exec.Command("systemctl", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w: %s", args[0], err, trimOutput(out))
	}
	return nil
}

func enableUserLinger() error {
	if _, err := exec.LookPath("loginctl"); err != nil {
		return nil
	}
	u := os.Getenv("USER")
	if u == "" {
		return nil
	}
	cmd := exec.Command("loginctl", "enable-linger", u)
	_ = cmd.Run()
	return nil
}

func printInstallSuccess(scope, serviceName, binary, config, unitPath string) {
	fmt.Println("arx-ca CA server installed and started.")
	fmt.Printf("Scope:    %s\n", scope)
	fmt.Printf("Service:  %s\n", serviceName)
	fmt.Printf("Unit:     %s\n", unitPath)
	fmt.Printf("Binary:   %s\n", binary)
	fmt.Printf("Config:   %s\n", config)
	fmt.Println("Edit server.toml (JWT secret, bootstrap password hash) before production use.")
}
