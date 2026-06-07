//go:build !windows

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// RestartExecutable replaces the current process with a fresh instance of the updated binary.
func RestartExecutable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable absolute path: %w", err)
	}
	if err := syscall.Exec(abs, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("exec updated binary: %w", err)
	}
	return nil
}
