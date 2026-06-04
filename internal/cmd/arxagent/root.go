package arxagentcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	updatecli "github.com/ARCOOON/arx-ca/internal/cli/update"
	"github.com/ARCOOON/arx-ca/internal/updater"
)

// Execute runs the arx-agent root command.
func Execute(version, commit string) error {
	return NewRootCmd(version, commit).Execute()
}

// NewRootCmd builds the lightweight arx-agent CLI (renewal daemon and local data-plane tools).
func NewRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "arx-agent",
		Short:   "Arx certificate renewal agent",
		Long:    "arx-agent is the lightweight data-plane binary for certificate renewal, local trust stores, and public certificate access on client nodes.",
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
		updatecli.NewCmd(updater.ComponentArxAgent),
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
