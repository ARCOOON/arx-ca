package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const configFileName = "server.yaml"

func (o InstallOptions) resolvedInstallDir() (string, error) {
	if dir := strings.TrimSpace(o.InstallDir); dir != "" {
		return filepath.Clean(dir), nil
	}
	if o.Scope.IsSystem() {
		return defaultSystemInstallDir(), nil
	}
	return defaultUserInstallDir()
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

func bootstrapConfig(installDir, destBinary string) error {
	configPath := filepath.Join(installDir, configFileName)
	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file %s: %w", configPath, err)
	}

	cmd := exec.Command(destBinary, "server", "config", "init")
	cmd.Dir = installDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bootstrap server config: %w: %s", err, trimOutput(out))
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
