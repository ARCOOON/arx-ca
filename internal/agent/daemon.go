package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cliapi "github.com/your-org/arx-ca/internal/cli/api"
	"github.com/your-org/arx-ca/internal/cli/runtime"
	"github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/models"
)

// RunDaemon blocks and periodically checks managed certificates, renewing them
// when their remaining TTL falls below the configured threshold.
func RunDaemon(cfg *config.AgentConfig) error {
	if cfg == nil {
		return fmt.Errorf("agent config is required")
	}

	checkInterval, err := cfg.Daemon.CheckIntervalDuration()
	if err != nil {
		return fmt.Errorf("parse daemon.check_interval: %w", err)
	}
	if checkInterval <= 0 {
		return fmt.Errorf("daemon.check_interval must be positive")
	}

	renewThreshold, err := cfg.Daemon.RenewThresholdDuration()
	if err != nil {
		return fmt.Errorf("parse daemon.renew_threshold: %w", err)
	}
	if renewThreshold <= 0 {
		return fmt.Errorf("daemon.renew_threshold must be positive")
	}

	if len(cfg.Daemon.ManagedCerts) == 0 {
		slog.Warn("daemon started with no managed certificates configured")
	}

	slog.Info("agent daemon started",
		"check_interval", checkInterval,
		"renew_threshold", renewThreshold,
		"managed_certs", len(cfg.Daemon.ManagedCerts),
	)

	runCheck(cfg, renewThreshold)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		runCheck(cfg, renewThreshold)
	}

	return nil
}

func runCheck(cfg *config.AgentConfig, renewThreshold time.Duration) {
	ctx := context.Background()

	client, err := runtime.NewAuthenticatedClient("")
	if err != nil {
		slog.Error("daemon check skipped: failed to build authenticated client", "error", err)
		return
	}

	for i, managed := range cfg.Daemon.ManagedCerts {
		if err := checkManagedCert(ctx, client, managed, renewThreshold); err != nil {
			slog.Error("managed certificate check failed",
				"index", i,
				"cert_path", managed.CertPath,
				"common_name", managed.CommonName,
				"error", err,
			)
		}
	}
}

func checkManagedCert(ctx context.Context, client *cliapi.Client, managed config.ManagedCert, renewThreshold time.Duration) error {
	certPath := strings.TrimSpace(managed.CertPath)
	keyPath := strings.TrimSpace(managed.KeyPath)
	commonName := strings.TrimSpace(managed.CommonName)

	if certPath == "" {
		return fmt.Errorf("cert_path is required")
	}
	if keyPath == "" {
		return fmt.Errorf("key_path is required")
	}
	if commonName == "" {
		return fmt.Errorf("common_name is required")
	}

	ttl, err := GetCertTTL(certPath)
	if err != nil {
		return err
	}

	slog.Debug("certificate TTL evaluated",
		"cert_path", certPath,
		"common_name", commonName,
		"ttl", ttl,
		"renew_threshold", renewThreshold,
	)

	if ttl >= renewThreshold {
		slog.Debug("certificate does not require renewal",
			"cert_path", certPath,
			"common_name", commonName,
			"ttl", ttl,
		)
		return nil
	}

	slog.Info("renewing certificate",
		"cert_path", certPath,
		"common_name", commonName,
		"ttl", ttl,
		"renew_threshold", renewThreshold,
	)

	req := models.AutoCertificateRequest{
		CommonName: commonName,
		DNSSANs:    []string{commonName},
		TemplateID: strings.TrimSpace(managed.Template),
	}

	resp, err := client.AutoCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("request certificate: %w", err)
	}
	if strings.TrimSpace(resp.CertificatePEM) == "" {
		return fmt.Errorf("renewal response did not include a certificate")
	}
	if strings.TrimSpace(resp.PrivateKeyPEM) == "" {
		return fmt.Errorf("renewal response did not include a private key")
	}

	if err := writePEMFile(certPath, resp.CertificatePEM); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}
	if err := writePEMFile(keyPath, resp.PrivateKeyPEM); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	slog.Info("certificate renewed",
		"cert_path", certPath,
		"key_path", keyPath,
		"common_name", commonName,
		"serial", resp.Serial,
	)

	hook := strings.TrimSpace(managed.PostHook)
	if hook == "" {
		return nil
	}

	if err := runPostHook(hook); err != nil {
		return fmt.Errorf("post-renewal hook failed: %w", err)
	}

	slog.Info("post-renewal hook executed", "common_name", commonName, "hook", hook)
	return nil
}

func writePEMFile(path, pemContent string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(pemContent), 0o600); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

func runPostHook(hook string) error {
	cmd := exec.Command("sh", "-c", hook)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute hook %q: %w", hook, err)
	}
	return nil
}
