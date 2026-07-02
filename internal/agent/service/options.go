package service

// InstallOptions configures self-install and uninstall of the arx-ca-agent systemd unit.
type InstallOptions struct {
	RunAsUser  string
	InstallDir string
}

const (
	defaultRunAsUser  = "arx-ca-agent"
	defaultInstallDir = "/opt/arx-ca-agent"
	binaryName        = "arx-ca-agent"
	configFileName    = "agent.yaml"
)

func (o InstallOptions) runAsUser() string {
	if o.RunAsUser != "" {
		return o.RunAsUser
	}
	return defaultRunAsUser
}

func (o InstallOptions) installDir() string {
	if o.InstallDir != "" {
		return o.InstallDir
	}
	return defaultInstallDir
}
