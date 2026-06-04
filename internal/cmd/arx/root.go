package arxcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Execute runs the arx root command.
func Execute(version, commit string) error {
	return NewRootCmd(version, commit).Execute()
}

// NewRootCmd builds the arx control-plane CLI (server, admin tools, and utilities).
func NewRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "arx",
		Short:   "Arx Certificate Authority control plane",
		Long:    "arx is the control-plane binary for the Arx CA API server, administration CLI, and operator utilities. Use arx-agent on client nodes for renewal and local certificate operations.",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	root.AddCommand(
		newServerCmd(),
		newLoginCmd(),
		newUICmd(),
		newCertCmd(),
		newUtilCmd(),
		utilHashCmd(),
	)

	applyCobraErrorSilence(root)
	return root
}

// applyCobraErrorSilence sets SilenceUsage and SilenceErrors on cmd and all descendants.
// Usage is shown only for explicit help or invalid arguments; errors are printed once in main.
func applyCobraErrorSilence(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	for _, sub := range cmd.Commands() {
		applyCobraErrorSilence(sub)
	}
}
