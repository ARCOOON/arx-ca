package arxcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ExecuteServer runs the arx-ca server root command.
func ExecuteServer(version, commit string) error {
	return NewServerRootCmd(version, commit).Execute()
}

// ExecuteCLI runs the arx-ca-cli administrative root command.
func ExecuteCLI(version, commit string) error {
	return NewCLIRootCmd(version, commit).Execute()
}

// NewServerRootCmd builds the arx-ca server binary (HTTP API, WebUI host, lifecycle).
func NewServerRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "arx-ca",
		Short:   "Arx Certificate Authority server",
		Long:    "arx-ca is the CA server binary: HTTP API, enrollment protocols, WebUI host, and local server lifecycle. Use arx-ca-cli for remote administration and arx-ca-agent on client nodes for renewal.",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	root.AddCommand(newServerCmd())

	applyCobraErrorSilence(root)
	return root
}

// NewCLIRootCmd builds the arx-ca-cli remote administration binary.
func NewCLIRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "arx-ca-cli",
		Short:   "Arx CA remote administration CLI",
		Long:    "arx-ca-cli is the operator CLI for authenticating to a remote arx-ca server, managing certificates, and running admin utilities. Run arx-ca on the CA host for server lifecycle.",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	root.AddCommand(
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
