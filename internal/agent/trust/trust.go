package trust

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/smallstep/truststore"
	agentapi "github.com/your-org/arx-ca/internal/agent/api"
	"github.com/your-org/arx-ca/internal/agent/state"
)

const trustPrefix = "arx-ca "

// InstallRoot fetches the Root CA from the server and installs it into local trust stores.
func InstallRoot(ctx context.Context, apiURL string) error {
	client, err := agentapi.NewClient(apiURL)
	if err != nil {
		return err
	}

	pem, err := client.FetchRootPEM(ctx)
	if err != nil {
		return fmt.Errorf("fetch root CA: %w", err)
	}

	if err := state.SaveRootPEM(pem); err != nil {
		return err
	}
	if err := state.SaveConfig(state.Config{APIURL: normalizeURL(apiURL)}); err != nil {
		return err
	}

	return installPEM(pem, "root")
}

// UninstallRoot removes the previously installed Root CA from local trust stores.
func UninstallRoot() error {
	pemBytes, err := state.LoadRootPEM()
	if err != nil {
		return err
	}
	if err := uninstallPEM(string(pemBytes)); err != nil {
		return err
	}
	return state.RemoveRootPEM()
}

// InstallIntermediate fetches the Intermediate CA from the server and installs it into local trust stores.
func InstallIntermediate(ctx context.Context, apiURL string) error {
	client, err := agentapi.NewClient(apiURL)
	if err != nil {
		return err
	}

	pem, err := client.FetchIntermediatePEM(ctx)
	if err != nil {
		return fmt.Errorf("fetch intermediate CA: %w", err)
	}

	if err := state.SaveIntermediatePEM(pem); err != nil {
		return err
	}
	if err := state.SaveConfig(state.Config{APIURL: normalizeURL(apiURL)}); err != nil {
		return err
	}

	return installPEM(pem, "intermediate")
}

// UninstallIntermediate removes the previously installed Intermediate CA from local trust stores.
func UninstallIntermediate() error {
	pemBytes, err := state.LoadIntermediatePEM()
	if err != nil {
		return err
	}
	if err := uninstallPEM(string(pemBytes)); err != nil {
		return err
	}
	return state.RemoveIntermediatePEM()
}

func installPEM(pem, label string) error {
	tmp, err := os.CreateTemp("", "arx-cert-service-"+label+"-*.pem")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(pem); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp certificate: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp certificate: %w", err)
	}

	opts := []truststore.Option{
		truststore.WithPrefix(trustPrefix),
		truststore.WithFirefox(),
	}

	if err := truststore.InstallFile(tmpPath, opts...); err != nil {
		return fmt.Errorf("install %s CA into trust stores: %w", label, err)
	}
	return nil
}

func uninstallPEM(pem string) error {
	tmp, err := os.CreateTemp("", "arx-cert-service-uninstall-*.pem")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(pem); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp certificate: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp certificate: %w", err)
	}

	opts := []truststore.Option{
		truststore.WithPrefix(trustPrefix),
		truststore.WithFirefox(),
	}

	if err := truststore.UninstallFile(tmpPath, opts...); err != nil {
		return fmt.Errorf("uninstall CA from trust stores: %w", err)
	}
	return nil
}

func normalizeURL(apiURL string) string {
	return strings.TrimRight(strings.TrimSpace(apiURL), "/")
}
