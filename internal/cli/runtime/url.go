package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	agentstate "github.com/your-org/arx-ca/internal/agent/state"
	clicfg "github.com/your-org/arx-ca/internal/cli/config"
	arxconfig "github.com/your-org/arx-ca/internal/config"
)

// ErrServerURLNotConfigured is returned when no server URL can be resolved.
var ErrServerURLNotConfigured = errors.New("Server URL not configured. Please run 'arx login --url <URL>' first, or provide the --url flag.")

// URLResolveOptions configures server URL resolution for CLI commands.
type URLResolveOptions struct {
	// FlagOverride is the value of the --url flag when set.
	FlagOverride string
	// PersistFlag when true writes a non-empty FlagOverride to local config files.
	PersistFlag bool
	// UseAgentState when true includes ~/.arx-cert-service/config.json api_url as a fallback.
	UseAgentState bool
}

// ResolveServerURL resolves the target server URL in priority order:
//  1. Non-empty --url flag (optionally persisted when PersistFlag is true)
//  2. server_url from ~/.arx/cli.yaml (Viper)
//  3. server_url from ~/.arx/config.json (saved at login)
//  4. api_url from ~/.arx-cert-service/config.json when UseAgentState is true
func ResolveServerURL(opts URLResolveOptions) (string, error) {
	if url := strings.TrimSpace(opts.FlagOverride); url != "" {
		normalized := normalizeURL(url)
		if opts.PersistFlag {
			if err := PersistServerURL(normalized, opts.UseAgentState); err != nil {
				return "", err
			}
		}
		return normalized, nil
	}

	if url := strings.TrimSpace(viper.GetString("server_url")); url != "" {
		return normalizeURL(url), nil
	}

	disk, err := clicfg.Load()
	if err != nil {
		return "", err
	}
	if url := strings.TrimSpace(disk.ServerURL); url != "" {
		return normalizeURL(url), nil
	}

	if opts.UseAgentState {
		agentCfg, err := agentstate.LoadConfig()
		if err != nil {
			return "", err
		}
		if url := strings.TrimSpace(agentCfg.APIURL); url != "" {
			return normalizeURL(url), nil
		}
	}

	return "", ErrServerURLNotConfigured
}

// PersistServerURL stores the server URL in ~/.arx/cli.yaml and ~/.arx/config.json.
// When includeAgentState is true, ~/.arx-cert-service/config.json is updated as well.
func PersistServerURL(url string, includeAgentState bool) error {
	url = normalizeURL(url)
	if url == "" {
		return fmt.Errorf("server url is empty")
	}

	if err := arxconfig.SetCLIServerURL(url); err != nil {
		return err
	}

	cfg, err := clicfg.Load()
	if err != nil {
		return err
	}
	cfg.ServerURL = url
	if err := clicfg.Save(cfg); err != nil {
		return err
	}

	if !includeAgentState {
		return nil
	}

	agentCfg, err := agentstate.LoadConfig()
	if err != nil {
		return err
	}
	agentCfg.APIURL = url
	return agentstate.SaveConfig(agentCfg)
}

func normalizeURL(url string) string {
	return strings.TrimRight(strings.TrimSpace(url), "/")
}
