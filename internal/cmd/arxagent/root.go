package arxagentcmd

import (
	"github.com/spf13/cobra"
)

// Execute runs the arx-agent root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds the lightweight arx-agent CLI (renewal daemon and local data-plane tools).
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arx-agent",
		Short: "Arx certificate renewal agent",
		Long:  "arx-agent is the lightweight data-plane binary for certificate renewal, local trust stores, and public certificate access on client nodes.",
	}

	root.AddCommand(
		newDaemonCmd(),
		newRunCmd(),
		newEnrollCmd(),
		newLocalCmd(),
		newTrustCmd(),
		newCertCmd(),
		newServiceCmd(),
	)

	return withCLIConfig(root)
}
