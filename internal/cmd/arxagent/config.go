package arxagentcmd

import (
	"log"

	"github.com/spf13/cobra"

	arxconfig "github.com/your-org/arx-ca/internal/config"
)

func newConfigCmd() *cobra.Command {
	config := &cobra.Command{
		Use:   "config",
		Short: "Manage agent configuration files",
	}

	init := &cobra.Command{
		Use:   "init",
		Short: "Generate a template agent.yaml configuration file",
		Long: `Writes agent.yaml with daemon defaults and example managed_certs entries
for both native API renewal and ACMEv2 client renewal.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFlag, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			path, err := arxconfig.ResolveAgentConfigPath(configFlag)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}
			if err := arxconfig.WriteAgentConfig(path, force); err != nil {
				return err
			}
			log.Printf("Configuration successfully generated at %s. Edit managed_certs before starting the daemon.", path)
			return nil
		},
	}
	init.Flags().String("config", "", "Path to agent.yaml (default: ~/.arx-cert-service/agent.yaml)")
	init.Flags().Bool("force", false, "Overwrite an existing configuration file")

	config.AddCommand(init)
	return config
}
