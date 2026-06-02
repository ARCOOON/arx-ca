package arxcmd

import (
	"github.com/spf13/cobra"
)

// Execute runs the arx root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds the arx control-plane CLI (server, admin tools, and utilities).
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arx",
		Short: "Arx Certificate Authority control plane",
		Long:  "arx is the control-plane binary for the Arx CA API server, administration CLI, and operator utilities. Use arx-agent on client nodes for renewal and local certificate operations.",
	}

	root.AddCommand(
		newServerCmd(),
		newLoginCmd(),
		newUICmd(),
		newCertCmd(),
		newUtilCmd(),
		utilHashCmd(),
	)

	return root
}
