package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/cli/runtime"
	"github.com/ARCOOON/arx-ca/internal/config"
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

	for i, managed := range cfg.Daemon.ManagedCerts {
		if err := managed.Validate(); err != nil {
			return fmt.Errorf("managed_certs[%d]: %w", i, err)
		}
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

	var apiRenewer *APIRenewer
	var acmeRenewer *ACMERenewer

	for i, managed := range cfg.Daemon.ManagedCerts {
		if err := checkManagedCert(ctx, managed, renewThreshold, &apiRenewer, &acmeRenewer); err != nil {
			slog.Error("managed certificate check failed",
				"index", i,
				"cert_path", managed.CertPath,
				"common_name", managed.CommonName,
				"protocol", managed.ProtocolName(),
				"error", err,
			)
		}
	}
}

func checkManagedCert(
	ctx context.Context,
	managed config.ManagedCert,
	renewThreshold time.Duration,
	apiRenewer **APIRenewer,
	acmeRenewer **ACMERenewer,
) error {
	certPath := strings.TrimSpace(managed.CertPath)
	commonName := strings.TrimSpace(managed.CommonName)

	ttl, err := certTTLForRenewal(certPath)
	if err != nil {
		return err
	}

	slog.Debug("certificate TTL evaluated",
		"cert_path", certPath,
		"common_name", commonName,
		"protocol", managed.ProtocolName(),
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
		"protocol", managed.ProtocolName(),
		"ttl", ttl,
		"renew_threshold", renewThreshold,
	)

	renewer, err := renewerForManaged(managed, apiRenewer, acmeRenewer)
	if err != nil {
		return err
	}

	if err := renewer.Renew(ctx, managed); err != nil {
		return err
	}

	slog.Info("certificate renewed",
		"cert_path", certPath,
		"key_path", managed.KeyPath,
		"common_name", commonName,
		"protocol", managed.ProtocolName(),
	)

	hook := strings.TrimSpace(managed.PostHook)
	if hook != "" {
		slog.Info("post-renewal hook executed", "common_name", commonName, "hook", hook)
	}

	return nil
}

func certTTLForRenewal(certPath string) (time.Duration, error) {
	ttl, err := GetCertTTL(certPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("certificate file not found; scheduling issuance", "cert_path", certPath)
			return 0, nil
		}
		return 0, err
	}
	return ttl, nil
}

func renewerForManaged(
	managed config.ManagedCert,
	apiRenewer **APIRenewer,
	acmeRenewer **ACMERenewer,
) (Renewer, error) {
	switch managed.ProtocolName() {
	case config.AgentProtocolACME:
		if *acmeRenewer == nil {
			*acmeRenewer = NewACMERenewer(nil)
		}
		return *acmeRenewer, nil
	default:
		if *apiRenewer == nil {
			client, err := runtime.NewAuthenticatedClient("")
			if err != nil {
				return nil, fmt.Errorf("build authenticated API client: %w", err)
			}
			*apiRenewer = NewAPIRenewer(client)
		}
		return *apiRenewer, nil
	}
}

// Ensure APIRenewer implements Renewer.
var _ Renewer = (*APIRenewer)(nil)

// Ensure ACMERenewer implements Renewer.
var _ Renewer = (*ACMERenewer)(nil)
