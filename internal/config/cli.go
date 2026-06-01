package config

// CLIConfig holds defaults for arx-ca-cli loaded from ~/.arx/cli.yaml.
type CLIConfig struct {
	ServerURL string `mapstructure:"server_url" yaml:"server_url"`
	LogLevel  string `mapstructure:"log_level" yaml:"log_level"`
}

// DefaultCLIConfig returns the built-in defaults used when cli.yaml is created.
func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		ServerURL: "http://localhost:8080",
		LogLevel:  "info",
	}
}
