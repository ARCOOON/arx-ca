package arxcmd

import (
	"github.com/spf13/cobra"

	updatecli "github.com/ARCOOON/arx-ca/internal/cli/update"
	"github.com/ARCOOON/arx-ca/internal/cli/util"
	"github.com/ARCOOON/arx-ca/internal/updater"
)

func newUtilCmd() *cobra.Command {
	utilCmd := &cobra.Command{
		Use:   "util",
		Short: "Administrative utility commands",
		Long:  "Helper commands for password hashing and other admin tasks.",
	}
	utilCmd.AddCommand(
		withCLIConfig(util.NewHashCmd()),
		updatecli.NewCmd(updater.ComponentArxCACli),
	)
	return utilCmd
}
