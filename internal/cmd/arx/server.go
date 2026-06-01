package arxcmd

import (
	"errors"
	"log"

	"github.com/spf13/cobra"

	arxconfig "github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/server/service"
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
	svc := &cobra.Command{
		Use:   "service",
		Short: "Install or remove the systemd unit for the Arx CA server",
	}

	install := &cobra.Command{
		Use:   "install",
		Short: "Install and enable the arx-ca-server systemd unit",
		RunE: func(_ *cobra.Command, _ []string) error {
			return service.Install(serverConfigFlag)
		},
	}

	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop, disable, and remove the arx-ca-server systemd unit",
		RunE: func(_ *cobra.Command, _ []string) error {
			return service.Uninstall()
		},
	}

	svc.AddCommand(install, uninstall)
	return svc
}

func skipsServerConfigInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "service", "config":
			return true
		}
	}
	return false
}
