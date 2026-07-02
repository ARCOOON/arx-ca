package arxagentcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	updatecli "github.com/ARCOOON/arx-ca/internal/cli/update"
	"github.com/ARCOOON/arx-ca/internal/updater"
)

// Execute runs the arx-ca-agent root command.
func Execute(version, commit string) error {
	return NewRootCmd(version, commit).Execute()
}

// NewRootCmd builds the arx-ca-agent renewal daemon and local data-plane tools.
func NewRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "arx-ca-agent",
		Short:   "Arx certificate renewal agent",
		Long:    "arx-ca-agent is the data-plane binary for automated certificate enrollment, renewal, local trust stores, and public certificate access on client nodes.",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	root.AddCommand(
		newConfigCmd(),
		newDaemonCmd(),
		newRunCmd(),
		newEnrollCmd(),
		newLocalCmd(),
		newTrustCmd(),
		newCertCmd(),
		newServiceCmd(),
		updatecli.NewCmd(updater.ComponentArxCAAgent),
	)

	applyCobraErrorSilence(root)
	return withCLIConfig(root)
}

// applyCobraErrorSilence sets SilenceUsage and SilenceErrors on cmd and all descendants.
func applyCobraErrorSilence(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	for _, sub := range cmd.Commands() {
		applyCobraErrorSilence(sub)
	}
}
