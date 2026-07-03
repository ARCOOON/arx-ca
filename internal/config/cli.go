package config

// CLIConfig holds defaults and persisted settings for arx-ca-cli (~/.arx-ca/cli.yaml).
// Authentication tokens are stored in ~/.arx-ca/config.json (see internal/cli/config.Config);
// both files share server_url after a successful login.
type CLIConfig struct {
	ServerURL string `mapstructure:"server_url" toml:"server_url,omitempty"`
	LogLevel  string `mapstructure:"log_level" toml:"log_level"`
	Token     string `mapstructure:"token" toml:"token,omitempty"`
	TokenType string `mapstructure:"token_type" toml:"token_type,omitempty"`
	Username  string `mapstructure:"username" toml:"username,omitempty"`
}

// DefaultCLIConfig returns the built-in defaults used when cli.yaml is created.
func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		LogLevel: "info",
	}
}
