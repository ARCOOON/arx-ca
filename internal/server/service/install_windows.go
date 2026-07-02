//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	windowsServiceName = "arx"
	windowsTaskName    = `ARX CA Server`
)

// Install registers the arx Windows service or user logon scheduled task for the selected scope.
func Install(opts InstallOptions) error {
	if opts.Scope.IsSystem() {
		return installWindowsSystem(opts)
	}
	return installWindowsUser(opts)
}

// Uninstall removes the Windows service or user scheduled task and install directory.
func Uninstall(opts InstallOptions) error {
	if opts.Scope.IsSystem() {
		return uninstallWindowsSystem(opts)
	}
	return uninstallWindowsUser(opts)
}

func installWindowsSystem(opts InstallOptions) error {
	if !isWindowsAdministrator() {
		return fmt.Errorf("service install with --system requires Administrator privileges")
	}
	if err := requireScExe(); err != nil {
		return err
	}

	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}
	destBinary := filepath.Join(installDir, binaryFileName())
	configPath := filepath.Join(installDir, configFileName)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
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

	if err := installWindowsService(destBinary, configPath); err != nil {
		return err
	}

	printInstallSuccess("system", windowsServiceName, destBinary, configPath, "Windows Service")
	return nil
}

func installWindowsUser(opts InstallOptions) error {
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

	if err := installUserLogonTask(destBinary, configPath); err != nil {
		return err
	}

	printInstallSuccess("user", windowsTaskName, destBinary, configPath, "Scheduled Task (ONLOGON)")
	fmt.Println("Windows does not support per-user Windows Services; the server starts at user logon via schtasks.")
	return nil
}

func uninstallWindowsSystem(opts InstallOptions) error {
	if !isWindowsAdministrator() {
		return fmt.Errorf("service uninstall with --system requires Administrator privileges")
	}

	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}

	_ = removeWindowsService()
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove install directory %s: %w", installDir, err)
	}

	fmt.Println("arx-ca CA server uninstalled (system scope).")
	return nil
}

func uninstallWindowsUser(opts InstallOptions) error {
	installDir, err := opts.resolvedInstallDir()
	if err != nil {
		return err
	}

	_ = removeUserLogonTask()
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove install directory %s: %w", installDir, err)
	}

	fmt.Println("arx-ca CA server uninstalled (user scope).")
	return nil
}

func isWindowsAdministrator() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	member, err := token.IsMember(sid)
	return err == nil && member
}

func requireScExe() error {
	if _, err := exec.LookPath("sc.exe"); err != nil {
		return fmt.Errorf("sc.exe is required for system service installation")
	}
	return nil
}

func installWindowsService(exePath, configPath string) error {
	_ = removeWindowsService()

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager: %w", err)
	}
	defer m.Disconnect()

	binPath := fmt.Sprintf(`"%s" server start --config "%s"`, exePath, configPath)
	s, err := m.CreateService(windowsServiceName, exePath, mgr.Config{
		DisplayName: "ARX Certificate Authority Server",
		Description: "ARX CA API server and certificate authority",
		StartType:   mgr.StartAutomatic,
	}, "server", "start", "--config", configPath)
	if err != nil {
		return fmt.Errorf("create Windows service %s (%s): %w", windowsServiceName, binPath, err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("start Windows service %s: %w", windowsServiceName, err)
	}
	return nil
}

func removeWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return nil
	}
	defer m.Disconnect()

	s, err := m.OpenService(windowsServiceName)
	if err != nil {
		return nil
	}
	defer s.Close()

	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		_, _ = s.Control(svc.Stop)
	}
	_ = s.Delete()
	return nil
}

func installUserLogonTask(exePath, configPath string) error {
	if _, err := exec.LookPath("schtasks.exe"); err != nil {
		return fmt.Errorf("schtasks.exe is required for user-scope installation")
	}

	_ = removeUserLogonTask()

	taskRun := fmt.Sprintf(`"%s" server start --config "%s"`, exePath, configPath)
	cmd := exec.Command(
		"schtasks.exe",
		"/Create",
		"/F",
		"/TN", windowsTaskName,
		"/TR", taskRun,
		"/SC", "ONLOGON",
		"/RL", "LIMITED",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create logon scheduled task: %w: %s", err, trimOutput(out))
	}
	return nil
}

func removeUserLogonTask() error {
	if _, err := exec.LookPath("schtasks.exe"); err != nil {
		return nil
	}
	cmd := exec.Command("schtasks.exe", "/Delete", "/F", "/TN", windowsTaskName)
	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(out), "ERROR: The system cannot find") {
		return fmt.Errorf("remove logon scheduled task: %w: %s", err, trimOutput(out))
	}
	return nil
}

func printInstallSuccess(scope, serviceName, binary, config, unitPath string) {
	fmt.Println("arx-ca CA server installed and started.")
	fmt.Printf("Scope:    %s\n", scope)
	fmt.Printf("Service:  %s\n", serviceName)
	fmt.Printf("Target:   %s\n", unitPath)
	fmt.Printf("Binary:   %s\n", binary)
	fmt.Printf("Config:   %s\n", config)
	fmt.Println("Edit server.yaml (JWT secret, bootstrap password hash) before production use.")
}
