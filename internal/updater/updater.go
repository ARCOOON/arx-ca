package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/Masterminds/semver/v3"
	"github.com/minio/selfupdate"

	"github.com/ARCOOON/arx-ca/internal/version"
)

// Component identifies which release artifact to download.
type Component string

const (
	ComponentArx      Component = "arx"
	ComponentArxAgent Component = "arx-agent"
)

// Config drives a single self-update run.
type Config struct {
	Component Component
	Channel   string
	Current   string
	Out       io.Writer
	Client    *http.Client
}

// Run checks GitHub for a newer release and applies an in-place binary update when available.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: requestTimeout}
	}
	current := strings.TrimSpace(cfg.Current)
	if current == "" {
		current = version.Current()
	}
	if err := validateComponent(cfg.Component); err != nil {
		return err
	}

	fmt.Fprintln(cfg.Out, "Checking for updates...")

	channel := strings.TrimSpace(strings.ToLower(cfg.Channel))
	if channel == "" {
		channel = "main"
	}
	release, err := fetchReleaseForChannel(ctx, cfg.Client, channel)
	if err != nil {
		return err
	}

	remoteTag := strings.TrimSpace(release.TagName)
	cmp, err := compareVersions(current, remoteTag)
	if err != nil {
		return fmt.Errorf("compare versions: %w", err)
	}
	if cmp <= 0 {
		display := normalizeTag(current)
		fmt.Fprintf(cfg.Out, "You are already running the latest version (%s)\n", display)
		return &AlreadyLatestError{Version: display}
	}

	fmt.Fprintf(cfg.Out, "New version %s found. Downloading...\n", normalizeTag(remoteTag))

	assetName := expectedAssetName(cfg.Component)
	downloadURL, err := findAssetURL(release, assetName)
	if err != nil {
		return err
	}

	if err := downloadAndApply(ctx, cfg.Client, downloadURL); err != nil {
		return err
	}

	fmt.Fprintf(cfg.Out, "Successfully updated to %s! Please restart the service.\n", normalizeTag(remoteTag))
	return nil
}

func validateComponent(c Component) error {
	switch c {
	case ComponentArx, ComponentArxAgent:
		return nil
	default:
		return fmt.Errorf("unsupported component %q", c)
	}
}

func expectedAssetName(component Component) string {
	name := fmt.Sprintf("%s-%s-%s", component, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func normalizeTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return version.Default
	}
	if !strings.HasPrefix(v, "v") && !strings.HasPrefix(v, "V") {
		return "v" + v
	}
	return v
}

func semverString(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" {
		return "0.0.0-dev"
	}
	return v
}

func compareVersions(current, remote string) (int, error) {
	curVer, err := semver.NewVersion(semverString(current))
	if err != nil {
		return 0, fmt.Errorf("parse current version %q: %w", current, err)
	}
	remVer, err := semver.NewVersion(semverString(remote))
	if err != nil {
		return 0, fmt.Errorf("parse remote version %q: %w", remote, err)
	}
	return remVer.Compare(curVer), nil
}

func downloadAndApply(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &NetworkError{Err: err}
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if isRateLimited(resp) {
			return &RateLimitError{
				Message: fmt.Sprintf("download rate limited (HTTP %d)", resp.StatusCode),
			}
		}
	}
	if resp.StatusCode != http.StatusOK {
		return &NetworkError{Err: fmt.Errorf("download failed: %s", resp.Status)}
	}

	if err := selfupdate.Apply(resp.Body, selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("update failed and rollback failed: %v (original: %w)", rerr, err)
		}
		if isPermissionError(err) {
			return &PermissionError{Err: err}
		}
		return fmt.Errorf("apply update: %w", err)
	}
	return nil
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if errors.Is(pathErr.Err, syscall.EACCES) || errors.Is(pathErr.Err, syscall.EPERM) {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "operation not permitted")
}
