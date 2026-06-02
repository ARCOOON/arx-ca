package arxagentcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/arx-ca/internal/agent"
	arxconfig "github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/logging"
)

func newDaemonCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run a long-lived renewal loop for managed certificates",
		Long:  "Monitors certificate files configured in agent.yaml, renews them through the CA API when their TTL falls below the configured threshold, and optionally runs post-renewal shell hooks.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAgentDaemon(configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "Path to agent.yaml (default: ~/.arx-cert-service/agent.yaml)")
	return cmd
}

func newRunCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the renewal daemon (alias for daemon)",
		Long:  "Same as arx-agent daemon. Intended for systemd ExecStart and production service units.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAgentDaemon(configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "Path to agent.yaml (default: ~/.arx-cert-service/agent.yaml)")
	return cmd
}

func runAgentDaemon(configPath string) error {
	logging.Configure(arxconfig.CLIConfigFromViper().LogLevel)

	if strings.TrimSpace(configPath) != "" {
		if err := arxconfig.SetAgentConfigPath(configPath); err != nil {
			return err
		}
	}
	if err := arxconfig.InitAgentConfig(); err != nil {
		return err
	}

	cfg := arxconfig.AgentConfigFromViper()
	return agent.RunDaemon(&cfg)
}
