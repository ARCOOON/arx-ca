package arxagentcmd

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	agentservice "github.com/ARCOOON/arx-ca/internal/agent/service"
)

func requireRootForService(action string) {
	if os.Geteuid() != 0 {
		log.Fatalf("%s must be executed as root", action)
	}
}

func newServiceCmd() *cobra.Command {
	var runAsUser, installDir string

	svc := &cobra.Command{
		Use:   "service",
		Short: "Install or remove the systemd unit for the arx-ca-agent daemon",
		Long: `Self-install the arx-ca-agent binary under /opt/arx-ca-agent (by default),
bootstrap agent.yaml, register arx-ca-agent.service, and start the renewal daemon. Requires root on Linux.`,
	}

	addServiceFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(&runAsUser, "run-as-user", "", "POSIX account that runs arx-ca-agent (default: arx-ca-agent)")
		cmd.Flags().StringVar(&installDir, "install-dir", "", "Install root for the binary and agent.yaml (default: /opt/arx-ca-agent)")
	}

	install := &cobra.Command{
		Use:   "install",
		Short: "Install the arx-ca-agent binary, configuration, and systemd unit",
		Run: func(cmd *cobra.Command, _ []string) {
			opts := resolveAgentServiceOptions(cmd, runAsUser, installDir)
			requireRootForService("service install")
			if err := agentservice.Install(opts); err != nil {
				log.Fatal(err)
			}
		},
	}
	addServiceFlags(install)

	uninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop the arx-ca-agent unit and remove the install directory",
		Run: func(cmd *cobra.Command, _ []string) {
			opts := resolveAgentServiceOptions(cmd, runAsUser, installDir)
			requireRootForService("service uninstall")
			if err := agentservice.Uninstall(opts); err != nil {
				log.Fatal(err)
			}
		},
	}
	addServiceFlags(uninstall)

	svc.AddCommand(install, uninstall)
	return svc
}

func resolveAgentServiceOptions(cmd *cobra.Command, flagRunAsUser, flagInstallDir string) agentservice.InstallOptions {
	opts := agentservice.InstallOptions{}

	if cmd.Flags().Changed("run-as-user") {
		opts.RunAsUser = strings.TrimSpace(flagRunAsUser)
	}
	if cmd.Flags().Changed("install-dir") {
		opts.InstallDir = strings.TrimSpace(flagInstallDir)
	}

	return opts
}
