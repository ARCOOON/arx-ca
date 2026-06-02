package config

import "time"

// ManagedCert describes a certificate file pair monitored and renewed by the agent daemon.
type ManagedCert struct {
	CertPath   string `mapstructure:"cert_path" yaml:"cert_path"`
	KeyPath    string `mapstructure:"key_path" yaml:"key_path"`
	Template   string `mapstructure:"template" yaml:"template"`
	CommonName string `mapstructure:"common_name" yaml:"common_name"`
	PostHook   string `mapstructure:"post_hook" yaml:"post_hook,omitempty"`
}

// AgentDaemonConfig controls the long-running certificate renewal loop.
type AgentDaemonConfig struct {
	CheckInterval  string        `mapstructure:"check_interval" yaml:"check_interval"`
	RenewThreshold string        `mapstructure:"renew_threshold" yaml:"renew_threshold"`
	ManagedCerts   []ManagedCert `mapstructure:"managed_certs" yaml:"managed_certs"`
}

// AgentConfig is the root configuration loaded from agent.yaml.
type AgentConfig struct {
	Daemon AgentDaemonConfig `mapstructure:"daemon" yaml:"daemon"`
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

// CheckIntervalDuration parses the daemon check interval string.
func (d AgentDaemonConfig) CheckIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(d.CheckInterval)
}

// RenewThresholdDuration parses the renewal TTL threshold string.
func (d AgentDaemonConfig) RenewThresholdDuration() (time.Duration, error) {
	return time.ParseDuration(d.RenewThreshold)
}
