package arxcmd

import (
	"github.com/spf13/cobra"

	"github.com/your-org/arx-ca/internal/cli/login"
)

func newLoginCmd() *cobra.Command {
	var (
		serverURL string
		email     string
		password  string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with admin credentials and store a JWT locally",
		RunE: func(_ *cobra.Command, _ []string) error {
			return login.Run(login.Options{
				ServerURL: serverURL,
				Email:     email,
				Password:  password,
			})
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Server URL (saved to config on successful login)")
	cmd.Flags().StringVar(&email, "email", "", "Admin email (skips prompt)")
	cmd.Flags().StringVar(&password, "password", "", "Admin password (skips prompt; use only in automation)")

	return withCLIConfig(cmd)
}
