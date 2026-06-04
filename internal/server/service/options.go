package service

// InstallOptions configures self-install and uninstall of the arx CA server daemon.
type InstallOptions struct {
	Scope      InstallScope
	RunAsUser  string
	InstallDir string
}

const (
	defaultRunAsUser = "arx-ca"
)

func (o InstallOptions) runAsUser() string {
	if o.RunAsUser != "" {
		return o.RunAsUser
	}
	return defaultRunAsUser
}
