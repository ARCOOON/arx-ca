package arxcmd

import (
	"fmt"
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

func resolveInstallScope(_ *cobra.Command, flagUser, flagSystem bool) (service.InstallScope, error) {
	if flagUser && flagSystem {
		return 0, fmt.Errorf("cannot specify both --user and --system")
	}
	if flagSystem {
		return service.InstallScopeSystem, nil
	}
	if flagUser {
		return service.InstallScopeUser, nil
	}
	return service.DefaultInstallScope(), nil
}

func runServerServiceInstall(scope service.InstallScope, opts service.InstallOptions) {
	opts.Scope = scope
	if err := service.Install(opts); err != nil {
		log.Fatal(err)
	}
}

func runServerServiceUninstall(scope service.InstallScope, opts service.InstallOptions) {
	opts.Scope = scope
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
