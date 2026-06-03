package arxcmd

import (
	"github.com/spf13/cobra"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
)

func withCLIConfig(cmd *cobra.Command) *cobra.Command {
	existing := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		if err := arxconfig.InitCLIConfig(); err != nil {
			return err
		}
		if existing != nil {
			return existing(c, args)
		}
		return nil
	}
	return cmd
}
