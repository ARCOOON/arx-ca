package arxcmd

import (
	"github.com/spf13/cobra"

	"github.com/ARCOOON/arx-ca/internal/cli/util"
)

func newUtilCmd() *cobra.Command {
	utilCmd := &cobra.Command{
		Use:   "util",
		Short: "Administrative utility commands",
		Long:  "Helper commands for password hashing and other admin tasks.",
	}
	utilCmd.AddCommand(withCLIConfig(util.NewHashCmd()))
	return utilCmd
}
