package arxcmd

import (
	"github.com/spf13/cobra"
)

// Execute runs the arx root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds the unified arx CLI (server, admin tools, and agent).
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arx",
		Short: "Arx Certificate Authority platform",
		Long:  "arx is the unified binary for the Arx CA API server, administration CLI, and local certificate agent.",
	}

	root.AddCommand(
		newServerCmd(),
		newLoginCmd(),
		newUICmd(),
		newCertCmd(),
		newUtilCmd(),
		utilHashCmd(),
		newAgentCmd(),
	)

	return root
}
