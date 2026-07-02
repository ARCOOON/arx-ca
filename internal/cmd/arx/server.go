package arxcmd

import (
	"errors"
	"log"

	"github.com/spf13/cobra"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

var serverConfigFlag string

func newServerCmd() *cobra.Command {
	server := &cobra.Command{
		Use:   "server",
		Short: "Run and manage the Arx CA API server",
		Long:  "Start the HTTP API and certificate authority, initialize configuration, or manage the systemd unit.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if skipsServerConfigInit(cmd) {
				return nil
			}
			if serverConfigFlag != "" {
				if err := arxconfig.SetServerConfigPath(serverConfigFlag); err != nil {
					return err
				}
			}
			if err := arxconfig.InitServerConfig(); err != nil {
				var notFound arxconfig.ServerConfigNotFoundError
				if errors.As(err, &notFound) {
					log.Fatal(err)
				}
				var persistErr *arxconfig.ErrConfigHealPersist
				if errors.As(err, &persistErr) {
					log.Fatal("cannot start server: failed to persist auto-secured configuration: ", persistErr)
				}
				return err
			}
			arxconfig.ApplyServerRuntimeFromViper()
			return nil
		},
	}

	server.PersistentFlags().StringVar(&serverConfigFlag, "config", "", "Path to server.yaml (default: server.yaml beside the executable)")

	server.AddCommand(
		newServerStartCmd(),
		newServerConfigCmd(),
		newServerSetupCmd(),
		newServerServiceCmd(),
		newServerUICmd(),
	)

	return server
}

func newServerStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the CA API server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServer()
		},
	}
}

func newServerConfigCmd() *cobra.Command {
	config := &cobra.Command{
		Use:   "config",
		Short: "Manage server configuration files",
	}

	init := &cobra.Command{
		Use:   "init",
		Short: "Generate a default server.yaml configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := arxconfig.ResolveServerConfigPath(serverConfigFlag)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}
			if err := arxconfig.WriteDefaultServerConfig(path, force); err != nil {
				return err
			}
			log.Printf("Configuration successfully generated at %s. Please edit it before starting the server.", path)
			return nil
		},
	}
	init.Flags().Bool("force", false, "Overwrite an existing configuration file")

	config.AddCommand(init)
	return config
}

func newServerServiceCmd() *cobra.Command {
	var runAsUser, installDir string
	var flagUser, flagSystem bool

	svc := &cobra.Command{
		Use:   "service",
		Short: "Install or remove the Arx CA server daemon",
		Long: `Self-install the arx-ca binary, bootstrap server.yaml, and register a daemon.

Linux: --system writes /etc/systemd/system/arx-ca.service (/opt/arx-ca by default).
       --user writes ~/.config/systemd/user/arx-ca.service ($HOME/.arx-ca by default).

Windows: --system registers a Windows Service under Program Files.
         --user creates a logon scheduled task under %LOCALAPPDATA%\\arx-ca.

When neither --user nor --system is set, user scope is used unless the process is root
(Linux) or Administrator (Windows), in which case system scope is selected.`,
	}

	addServiceFlags := func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(&flagUser, "user", false, "Install for the current user (default when non-privileged)")
		cmd.Flags().BoolVar(&flagSystem, "system", false, "Install system-wide (requires root on Linux or Administrator on Windows)")
		cmd.Flags().StringVar(&runAsUser, "run-as-user", "", "POSIX account for --system on Linux (overrides server.yaml service.run_as_user)")
		cmd.Flags().StringVar(&installDir, "install-dir", "", "Install root for binary and server.yaml (overrides server.yaml service.install_dir)")
	}

	install := &cobra.Command{
		Use:   "install",
		Short: "Install the arx-ca binary, configuration, and service unit",
		Run: func(cmd *cobra.Command, _ []string) {
			scope, err := resolveInstallScope(cmd, flagUser, flagSystem)
			if err != nil {
				log.Fatal(err)
			}
			opts, err := resolveServiceInstallOptions(cmd, runAsUser, installDir)
			if err != nil {
				log.Fatal(err)
			}
			runServerServiceInstall(scope, opts)
		},
	}
	addServiceFlags(install)

	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop the service unit and remove the install directory",
		Run: func(cmd *cobra.Command, _ []string) {
			scope, err := resolveInstallScope(cmd, flagUser, flagSystem)
			if err != nil {
				log.Fatal(err)
			}
			opts, err := resolveServiceInstallOptions(cmd, runAsUser, installDir)
			if err != nil {
				log.Fatal(err)
			}
			runServerServiceUninstall(scope, opts)
		},
	}
	addServiceFlags(uninstall)

	svc.AddCommand(install, uninstall)
	return svc
}

func skipsServerConfigInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "service", "config", "setup", "ui":
			return true
		}
	}
	return false
}
