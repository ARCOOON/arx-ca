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

	svc := &cobra.Command{
		Use:   "service",
		Short: "Install or remove the systemd unit for the Arx CA server",
		Long: `Self-install the arx binary under /opt/arx (by default), bootstrap server.yaml,
register a hardened arx-server systemd unit, and start the CA API. Requires root on Linux.`,
	}

	addServiceFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(&runAsUser, "run-as-user", "", "POSIX account that runs the arx-server service (overrides server.yaml service.run_as_user)")
		cmd.Flags().StringVar(&installDir, "install-dir", "", "Install root for the binary and server.yaml (overrides server.yaml service.install_dir)")
	}

	install := &cobra.Command{
		Use:   "install",
		Short: "Install the arx binary, configuration, and arx-server systemd unit",
		Run: func(cmd *cobra.Command, _ []string) {
			opts, err := resolveServiceInstallOptions(cmd, runAsUser, installDir)
			if err != nil {
				log.Fatal(err)
			}
			runServerServiceInstall(opts)
		},
	}
	addServiceFlags(install)

	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop the arx-server unit and remove the install directory",
		Run: func(cmd *cobra.Command, _ []string) {
			opts, err := resolveServiceInstallOptions(cmd, runAsUser, installDir)
			if err != nil {
				log.Fatal(err)
			}
			runServerServiceUninstall(opts)
		},
	}
	addServiceFlags(uninstall)

	svc.AddCommand(install, uninstall)
	return svc
}

func skipsServerConfigInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "service", "config", "setup":
			return true
		}
	}
	return false
}
