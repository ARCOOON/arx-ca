package updater

import (
	"errors"
	"fmt"
)

// Exit codes returned by ExitCode for scripting and automation.
const (
	ExitSuccess       = 0
	ExitAlreadyLatest = 0
	ExitGeneric       = 1
	ExitNetwork       = 2
	ExitRateLimit     = 3
	ExitPermission    = 4
)

// ErrAlreadyLatest indicates the running binary is up to date.
var ErrAlreadyLatest = errors.New("already running the latest version")

// AlreadyLatestError carries the version string for user-facing messages.
type AlreadyLatestError struct {
	Version string
}

func (e *AlreadyLatestError) Error() string {
	return fmt.Sprintf("already running the latest version (%s)", e.Version)
}

func (e *AlreadyLatestError) Is(target error) bool {
	return target == ErrAlreadyLatest
}

// NetworkError wraps connectivity and HTTP transport failures.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// RateLimitError indicates GitHub API rate limiting.
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "GitHub API rate limit exceeded; try again later or authenticate API requests"
}

// PermissionError indicates the binary could not be replaced (e.g. insufficient privileges).
type PermissionError struct {
	Err error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied updating binary: %v", e.Err)
}

func (e *PermissionError) Unwrap() error {
	return e.Err
}

// ExitCode maps updater errors to process exit codes.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var already *AlreadyLatestError
	if errors.As(err, &already) {
		return ExitAlreadyLatest
	}
	var net *NetworkError
	if errors.As(err, &net) {
		return ExitNetwork
	}
	var rate *RateLimitError
	if errors.As(err, &rate) {
		return ExitRateLimit
	}
	var perm *PermissionError
	if errors.As(err, &perm) {
		return ExitPermission
	}
	if errors.Is(err, ErrAlreadyLatest) {
		return ExitAlreadyLatest
	}
	return ExitGeneric
}
