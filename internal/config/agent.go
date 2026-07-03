package config

import (
	"fmt"
	"strings"
	"time"
)

const (
	AgentProtocolAPI  = "api"
	AgentProtocolACME = "acme"

	AgentChallengeHTTP01 = "http-01"
)

// ManagedCert describes a certificate file pair monitored and renewed by the agent daemon.
type ManagedCert struct {
	Protocol string `mapstructure:"protocol" toml:"protocol,omitempty"`

	CertPath   string `mapstructure:"cert_path" toml:"cert_path"`
	KeyPath    string `mapstructure:"key_path" toml:"key_path"`
	Template   string `mapstructure:"template" toml:"template,omitempty"`
	CommonName string `mapstructure:"common_name" toml:"common_name"`
	PostHook   string `mapstructure:"post_hook" toml:"post_hook,omitempty"`

	ACMEDirectoryURL    string `mapstructure:"acme_directory_url" toml:"acme_directory_url,omitempty"`
	ACMEEmail           string `mapstructure:"acme_email" toml:"acme_email,omitempty"`
	ChallengeType       string `mapstructure:"challenge_type" toml:"challenge_type,omitempty"`
	Webroot             string `mapstructure:"webroot" toml:"webroot,omitempty"`
	ChallengeListenPort int    `mapstructure:"challenge_listen_port" toml:"challenge_listen_port,omitempty"`
}

// AgentDaemonConfig controls the long-running certificate renewal loop.
type AgentDaemonConfig struct {
	CheckInterval  string        `mapstructure:"check_interval" toml:"check_interval"`
	RenewThreshold string        `mapstructure:"renew_threshold" toml:"renew_threshold"`
	ManagedCerts   []ManagedCert `mapstructure:"managed_certs" toml:"managed_certs"`
}

// AgentConfig is the root configuration loaded from agent.yaml.
type AgentConfig struct {
	Daemon AgentDaemonConfig `mapstructure:"daemon" toml:"daemon"`
}

// DefaultAgentConfig returns the built-in defaults used when agent.yaml is created.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Daemon: AgentDaemonConfig{
			CheckInterval:  "24h",
			RenewThreshold: "720h",
			ManagedCerts:   nil,
		},
	}
}

// TemplateAgentConfig returns a starter configuration with API and ACME examples.
func TemplateAgentConfig() AgentConfig {
	return AgentConfig{
		Daemon: AgentDaemonConfig{
			CheckInterval:  "24h",
			RenewThreshold: "720h",
			ManagedCerts: []ManagedCert{
				{
					Protocol:   AgentProtocolAPI,
					CertPath:   "/etc/nginx/ssl/app.pem",
					KeyPath:    "/etc/nginx/ssl/app-key.pem",
					Template:   "web-server",
					CommonName: "app.internal.example",
					PostHook:   "systemctl reload nginx",
				},
				{
					Protocol:            AgentProtocolACME,
					CertPath:            "/etc/nginx/ssl/acme-app.pem",
					KeyPath:             "/etc/nginx/ssl/acme-app-key.pem",
					CommonName:          "app.example.com",
					ACMEDirectoryURL:    "https://ca.example.com/acme/directory",
					ACMEEmail:           "admin@example.com",
					ChallengeType:       AgentChallengeHTTP01,
					Webroot:             "/var/www/html",
					ChallengeListenPort: 0,
					PostHook:            "systemctl reload nginx",
				},
			},
		},
	}
}

// ProtocolName returns the normalized renewal protocol ("api" or "acme").
func (m ManagedCert) ProtocolName() string {
	switch strings.ToLower(strings.TrimSpace(m.Protocol)) {
	case AgentProtocolACME:
		return AgentProtocolACME
	default:
		return AgentProtocolAPI
	}
}

// ChallengeTypeName returns the normalized ACME challenge type.
func (m ManagedCert) ChallengeTypeName() string {
	ch := strings.ToLower(strings.TrimSpace(m.ChallengeType))
	if ch == "" {
		return AgentChallengeHTTP01
	}
	return ch
}

// Validate checks managed certificate settings for the configured protocol.
func (m ManagedCert) Validate() error {
	certPath := strings.TrimSpace(m.CertPath)
	keyPath := strings.TrimSpace(m.KeyPath)
	commonName := strings.TrimSpace(m.CommonName)

	if certPath == "" {
		return fmt.Errorf("cert_path is required")
	}
	if keyPath == "" {
		return fmt.Errorf("key_path is required")
	}
	if commonName == "" {
		return fmt.Errorf("common_name is required")
	}

	switch m.ProtocolName() {
	case AgentProtocolAPI:
		return nil
	case AgentProtocolACME:
		if strings.TrimSpace(m.ACMEDirectoryURL) == "" {
			return fmt.Errorf("acme_directory_url is required when protocol is acme")
		}
		if strings.TrimSpace(m.ACMEEmail) == "" {
			return fmt.Errorf("acme_email is required when protocol is acme")
		}
		ch := m.ChallengeTypeName()
		if ch != AgentChallengeHTTP01 {
			return fmt.Errorf("unsupported challenge_type %q (only http-01 is supported)", ch)
		}
		webroot := strings.TrimSpace(m.Webroot)
		if webroot == "" && m.ChallengeListenPort <= 0 {
			return fmt.Errorf("either webroot or challenge_listen_port must be set for http-01")
		}
		if webroot != "" && m.ChallengeListenPort > 0 {
			return fmt.Errorf("webroot and challenge_listen_port are mutually exclusive")
		}
		return nil
	default:
		return fmt.Errorf("unsupported protocol %q (use api or acme)", m.Protocol)
	}
}

// CheckIntervalDuration parses the daemon check interval string.
func (d AgentDaemonConfig) CheckIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(d.CheckInterval)
}

// RenewThresholdDuration parses the renewal TTL threshold string.
func (d AgentDaemonConfig) RenewThresholdDuration() (time.Duration, error) {
	return time.ParseDuration(d.RenewThreshold)
}
