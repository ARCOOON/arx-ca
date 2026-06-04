package updatecli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ARCOOON/arx-ca/internal/updater"
	"github.com/ARCOOON/arx-ca/internal/version"
)

// NewCmd returns a cobra command that self-updates the running binary.
func NewCmd(component updater.Component) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Download and install the latest release from GitHub",
		Long: `Fetches the latest release from ARCOOON/arx-ca on GitHub, compares semantic
versions, and atomically replaces the running binary when a newer build is available.

Requires outbound HTTPS access to api.github.com and github.com. On Linux, run with
sufficient privileges to overwrite the executable (e.g. sudo when installed system-wide).`,
		Run: func(cmd *cobra.Command, _ []string) {
			ctx := cmd.Context()
			err := updater.Run(ctx, updater.Config{
				Component: component,
				Current:   version.Current(),
				Out:       os.Stdout,
			})
			if err == nil {
				return
			}
			var already *updater.AlreadyLatestError
			if errors.As(err, &already) {
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(updater.ExitCode(err))
		},
	}
}
