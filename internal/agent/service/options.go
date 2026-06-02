package service

// InstallOptions configures self-install and uninstall of the arx-agent systemd unit.
type InstallOptions struct {
	RunAsUser  string
	InstallDir string
}

const (
	defaultRunAsUser  = "arx-agent"
	defaultInstallDir = "/opt/arx-agent"
	binaryName        = "arx-agent"
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
