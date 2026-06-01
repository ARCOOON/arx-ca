package arxcmd

import (
	"github.com/spf13/cobra"

	"github.com/your-org/arx-ca/internal/cli/runtime"
	"github.com/your-org/arx-ca/internal/cli/tui"
)

func newUICmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive terminal UI",
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := runtime.NewAuthenticatedClient(serverURL)
			if err != nil {
				return err
			}
			return tui.Run(client)
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL (saved to config when set)")

	return withCLIConfig(cmd)
}
