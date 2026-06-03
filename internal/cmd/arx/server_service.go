package arxcmd

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/server/service"
)

func requireRootForService(action string) {
	if os.Geteuid() != 0 {
		log.Fatalf("%s must be executed as root", action)
	}
}

func runServerServiceInstall(opts service.InstallOptions) {
	requireRootForService("service install")
	if err := service.Install(opts); err != nil {
		log.Fatal(err)
	}
}

func runServerServiceUninstall(opts service.InstallOptions) {
	requireRootForService("service uninstall")
	if err := service.Uninstall(opts); err != nil {
		log.Fatal(err)
	}
}

// resolveServiceInstallOptions applies flag > server.yaml service block > hardcoded defaults.
func resolveServiceInstallOptions(cmd *cobra.Command, flagRunAsUser, flagInstallDir string) (service.InstallOptions, error) {
	opts := service.InstallOptions{}

	cfgUser, cfgDir, err := arxconfig.ServiceInstallSettingsFromConfig(serverConfigFlag)
	if err != nil {
		return service.InstallOptions{}, err
	}

	if cmd.Flags().Changed("run-as-user") {
		opts.RunAsUser = strings.TrimSpace(flagRunAsUser)
	} else {
		opts.RunAsUser = cfgUser
	}

	if cmd.Flags().Changed("install-dir") {
		opts.InstallDir = strings.TrimSpace(flagInstallDir)
	} else {
		opts.InstallDir = cfgDir
	}

	return opts, nil
}
