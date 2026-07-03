package models

import (
	"time"

	"github.com/ARCOOON/arx-ca/internal/config"
)

// SettingsConfigResponse is the masked server.toml view returned to the WebUI.
type SettingsConfigResponse struct {
	Server      ServerSettingsView    `json:"server"`
	Database    DatabaseConfigView    `json:"database"`
	CA          CAConfigView          `json:"ca"`
	CABootstrap CABootstrapConfigView `json:"ca_bootstrap"`
	Security    SecurityConfigView    `json:"security"`
	Bootstrap   BootstrapView         `json:"bootstrap"`
	Telemetry   TelemetryConfigView   `json:"telemetry"`
	Service     ServiceConfigView     `json:"service"`
	WebUI       WebUIConfigView       `json:"webui"`
	Updater     UpdaterConfigView     `json:"updater"`
}

// ServerSettingsView mirrors server.toml server settings for JSON APIs.
type ServerSettingsView struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	LogLevel     string        `json:"log_level"`
	ReadTimeout  string        `json:"read_timeout"`
	WriteTimeout string        `json:"write_timeout"`
	TLS          ServerTLSView `json:"tls"`
}

// ServerTLSView holds API TLS settings exposed to clients.
type ServerTLSView struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

// DatabaseConfigView mirrors database settings with secrets redacted.
type DatabaseConfigView struct {
	Driver       string `json:"driver"`
	Path         string `json:"path"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	PasswordFile string `json:"password_file"`
	DBName       string `json:"dbname"`
	SSLMode      string `json:"sslmode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

// CAConfigView mirrors CA integration settings with secrets redacted.
type CAConfigView struct {
	StepCAURL               string                   `json:"stepca_url"`
	RootPath                string                   `json:"root_path"`
	IntermediatePath        string                   `json:"intermediate_path"`
	ProvisionerName         string                   `json:"provisioner_name"`
	MaxTTL                  string                   `json:"max_ttl"`
	Password                string                   `json:"password"`
	PasswordFile            string                   `json:"password_file"`
	ProvisionerPasswordFile string                   `json:"provisioner_password_file"`
	Provisioners            CAProvisionersConfigView `json:"provisioners"`
}

// CAProvisionersConfigView groups enrollment provisioner settings.
type CAProvisionersConfigView struct {
	ACME ACMEProvisionerConfigView `json:"acme"`
	SCEP SCEPProvisionerConfigView `json:"scep"`
}

// ACMEProvisionerConfigView exposes ACME provisioner settings.
type ACMEProvisionerConfigView struct {
	Enabled           *bool    `json:"enabled"`
	RequireEAB        bool     `json:"require_eab"`
	Challenges        []string `json:"challenges"`
	DeviceAttestation bool     `json:"device_attestation"`
}

// SCEPProvisionerConfigView exposes SCEP provisioner settings.
type SCEPProvisionerConfigView struct {
	Enabled           *bool  `json:"enabled"`
	DeviceAttestation bool   `json:"device_attestation"`
	ChallengePassword string `json:"challenge_password"`
}

// CABootstrapConfigView mirrors first-run CA bootstrap parameters.
type CABootstrapConfigView struct {
	RootCN         string `json:"root_cn"`
	IntermediateCN string `json:"intermediate_cn"`
	Organization   string `json:"organization"`
	Country        string `json:"country"`
	KeySize        int    `json:"key_size"`
}

// SecurityConfigView mirrors authentication settings with secrets redacted.
type SecurityConfigView struct {
	JWTSecret            string `json:"jwt_secret"`
	TokenExpirationHours int    `json:"token_expiration_hours"`
	CookieSameSite       string `json:"cookie_same_site"`
	CookieSecure         *bool  `json:"cookie_secure"`
}

// BootstrapView mirrors bootstrap admin credentials with secrets redacted.
type BootstrapView struct {
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

// TelemetryConfigView mirrors OpenTelemetry exporter settings.
type TelemetryConfigView struct {
	ServiceName      string `json:"service_name"`
	ExporterEndpoint string `json:"exporter_endpoint"`
	ExporterInsecure bool   `json:"exporter_insecure"`
	SDKDisabled      bool   `json:"sdk_disabled"`
}

// ServiceConfigView mirrors systemd install parameters.
type ServiceConfigView struct {
	RunAsUser  string `json:"run_as_user"`
	InstallDir string `json:"install_dir"`
}

// WebUIConfigView mirrors WebUI listener settings.
type WebUIConfigView struct {
	Enabled       bool          `json:"enabled"`
	UIDir         string        `json:"ui_dir"`
	PathPrefix    string        `json:"path_prefix"`
	ListenAddress string        `json:"listen_address"`
	ProxyAPI      *bool         `json:"proxy_api"`
	MaxBodySize   int64         `json:"max_body_size"`
	ReadTimeout   string        `json:"read_timeout"`
	WriteTimeout  string        `json:"write_timeout"`
	TLS           WebUITLSView  `json:"tls"`
	CORS          WebUICORSView `json:"cors"`
}

// WebUITLSView holds WebUI TLS settings.
type WebUITLSView struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

// WebUICORSView holds WebUI CORS policy.
type WebUICORSView struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
}

// UpdaterConfigView mirrors background updater settings.
type UpdaterConfigView struct {
	Enabled                  bool   `json:"enabled"`
	Channel                  string `json:"channel"`
	NotifyOnly               bool   `json:"notify_only"`
	CheckInterval            string `json:"check_interval"`
	ViewChangelogAfterUpdate bool   `json:"view_changelog_after_update"`
}

// NewSettingsConfigResponse converts a masked ServerConfig into the API response shape.
func NewSettingsConfigResponse(cfg config.ServerConfig) SettingsConfigResponse {
	return SettingsConfigResponse{
		Server: ServerSettingsView{
			Host:         cfg.Server.Host,
			Port:         cfg.Server.Port,
			LogLevel:     cfg.Server.LogLevel,
			ReadTimeout:  durationString(cfg.Server.ReadTimeout),
			WriteTimeout: durationString(cfg.Server.WriteTimeout),
			TLS: ServerTLSView{
				Enabled:  cfg.Server.TLS.Enabled,
				CertFile: cfg.Server.TLS.CertFile,
				KeyFile:  cfg.Server.TLS.KeyFile,
			},
		},
		Database: DatabaseConfigView{
			Driver:       cfg.Database.Driver,
			Path:         cfg.Database.Path,
			Host:         cfg.Database.Host,
			Port:         cfg.Database.Port,
			User:         cfg.Database.User,
			Password:     cfg.Database.Password,
			PasswordFile: cfg.Database.PasswordFile,
			DBName:       cfg.Database.DBName,
			SSLMode:      cfg.Database.SSLMode,
			MaxOpenConns: cfg.Database.MaxOpenConns,
			MaxIdleConns: cfg.Database.MaxIdleConns,
		},
		CA: CAConfigView{
			StepCAURL:               cfg.CA.StepCAURL,
			RootPath:                cfg.CA.RootPath,
			IntermediatePath:        cfg.CA.IntermediatePath,
			ProvisionerName:         cfg.CA.ProvisionerName,
			MaxTTL:                  cfg.CA.MaxTTL,
			Password:                cfg.CA.Password,
			PasswordFile:            cfg.CA.PasswordFile,
			ProvisionerPasswordFile: cfg.CA.ProvisionerPasswordFile,
			Provisioners: CAProvisionersConfigView{
				ACME: ACMEProvisionerConfigView{
					Enabled:           cfg.CA.Provisioners.ACME.Enabled,
					RequireEAB:        cfg.CA.Provisioners.ACME.RequireEAB,
					Challenges:        append([]string(nil), cfg.CA.Provisioners.ACME.Challenges...),
					DeviceAttestation: cfg.CA.Provisioners.ACME.DeviceAttestation,
				},
				SCEP: SCEPProvisionerConfigView{
					Enabled:           cfg.CA.Provisioners.SCEP.Enabled,
					DeviceAttestation: cfg.CA.Provisioners.SCEP.DeviceAttestation,
					ChallengePassword: cfg.CA.Provisioners.SCEP.ChallengePassword,
				},
			},
		},
		CABootstrap: CABootstrapConfigView{
			RootCN:         cfg.CABootstrap.RootCN,
			IntermediateCN: cfg.CABootstrap.IntermediateCN,
			Organization:   cfg.CABootstrap.Organization,
			Country:        cfg.CABootstrap.Country,
			KeySize:        cfg.CABootstrap.KeySize,
		},
		Security: SecurityConfigView{
			JWTSecret:            cfg.Security.JWTSecret,
			TokenExpirationHours: cfg.Security.TokenExpirationHours,
			CookieSameSite:       cfg.Security.CookieSameSite,
			CookieSecure:         cfg.Security.CookieSecure,
		},
		Bootstrap: BootstrapView{
			AdminEmail:    cfg.Bootstrap.AdminEmail,
			AdminPassword: cfg.Bootstrap.AdminPassword,
		},
		Telemetry: TelemetryConfigView{
			ServiceName:      cfg.Telemetry.ServiceName,
			ExporterEndpoint: cfg.Telemetry.ExporterEndpoint,
			ExporterInsecure: cfg.Telemetry.ExporterInsecure,
			SDKDisabled:      cfg.Telemetry.SDKDisabled,
		},
		Service: ServiceConfigView{
			RunAsUser:  cfg.Service.RunAsUser,
			InstallDir: cfg.Service.InstallDir,
		},
		WebUI: WebUIConfigView{
			Enabled:       cfg.WebUI.Enabled,
			UIDir:         cfg.WebUI.UIDir,
			PathPrefix:    cfg.WebUI.PathPrefix,
			ListenAddress: cfg.WebUI.ListenAddress,
			ProxyAPI:      cfg.WebUI.ProxyAPI,
			MaxBodySize:   cfg.WebUI.MaxBodySize,
			ReadTimeout:   cfg.WebUI.ReadTimeout,
			WriteTimeout:  cfg.WebUI.WriteTimeout,
			TLS: WebUITLSView{
				Enabled:  cfg.WebUI.TLS.Enabled,
				CertFile: cfg.WebUI.TLS.CertFile,
				KeyFile:  cfg.WebUI.TLS.KeyFile,
			},
			CORS: WebUICORSView{
				AllowedOrigins:   append([]string(nil), cfg.WebUI.CORS.AllowedOrigins...),
				AllowedMethods:   append([]string(nil), cfg.WebUI.CORS.AllowedMethods...),
				AllowedHeaders:   append([]string(nil), cfg.WebUI.CORS.AllowedHeaders...),
				AllowCredentials: cfg.WebUI.CORS.AllowCredentials,
			},
		},
		Updater: UpdaterConfigView{
			Enabled:                  cfg.Updater.Enabled,
			Channel:                  cfg.Updater.Channel,
			NotifyOnly:               cfg.Updater.NotifyOnly,
			CheckInterval:            cfg.Updater.CheckInterval,
			ViewChangelogAfterUpdate: cfg.Updater.ViewChangelogAfterUpdate,
		},
	}
}

func durationString(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return d.String()
}
