package config

import "fmt"

// Bootstrap holds first-run admin credentials seeded when the users table is empty.
type Bootstrap struct {
	AdminEmail    string `mapstructure:"admin_email" yaml:"admin_email"`
	AdminPassword string `mapstructure:"admin_password" yaml:"admin_password"`
}

// ServerConfig holds runtime settings for arx-ca-server loaded from server.yaml.
type ServerConfig struct {
	Host                 string    `mapstructure:"host" yaml:"host"`
	Port                 int       `mapstructure:"port" yaml:"port"`
	CAConfigPath         string    `mapstructure:"ca_config_path" yaml:"ca_config_path"`
	LogLevel             string    `mapstructure:"log_level" yaml:"log_level"`
	DBType               string    `mapstructure:"db_type" yaml:"db_type"`
	DBDataSource         string    `mapstructure:"db_data_source" yaml:"db_data_source"`
	Bootstrap            Bootstrap `mapstructure:"bootstrap" yaml:"bootstrap"`
	OTELServiceName      string    `mapstructure:"otel_service_name" yaml:"otel_service_name"`
	OTELExporterEndpoint string    `mapstructure:"otel_exporter_endpoint" yaml:"otel_exporter_endpoint"`
	OTELExporterInsecure bool      `mapstructure:"otel_exporter_insecure" yaml:"otel_exporter_insecure"`
	OTELSDKDisabled      bool      `mapstructure:"otel_sdk_disabled" yaml:"otel_sdk_disabled"`
}

// DefaultServerConfig returns the built-in defaults used when server.yaml is created.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:                 "",
		Port:                 8080,
		CAConfigPath:         ".pki/config/ca.json",
		LogLevel:             "info",
		DBType:       "badgerv2",
		DBDataSource: "",
		Bootstrap: Bootstrap{
			AdminEmail:    "admin@arx.local",
			AdminPassword: "changeme",
		},
		OTELServiceName: "arx-ca",
		OTELExporterEndpoint: "http://localhost:4318",
		OTELExporterInsecure: true,
		OTELSDKDisabled:      false,
	}
}

// ListenAddress returns the HTTP listen address in host:port form (e.g. ":8080").
func (c ServerConfig) ListenAddress() string {
	if c.Port <= 0 {
		c.Port = DefaultServerConfig().Port
	}
	if c.Host == "" {
		return fmt.Sprintf(":%d", c.Port)
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
