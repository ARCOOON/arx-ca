package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cliapi "github.com/your-org/arx-ca/internal/cli/api"
	clicfg "github.com/your-org/arx-ca/internal/cli/config"
	"github.com/your-org/arx-ca/internal/cli/login"
	"github.com/your-org/arx-ca/internal/cli/tui"
	cliutil "github.com/your-org/arx-ca/internal/cli/util"
	arxconfig "github.com/your-org/arx-ca/internal/config"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arx-ca-cli",
		Short: "Super Admin CLI and terminal UI for arx-ca-server",
		Long:  "arx-ca-cli authenticates against arx-ca-server and provides a terminal UI for CA management and operations dashboards.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return arxconfig.InitCLIConfig()
		},
	}
	root.AddCommand(newLoginCmd())
	root.AddCommand(newUICmd())
	root.AddCommand(newUtilCmd())
	return root
}

func newUtilCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "util",
		Short: "Utility commands for arx-ca administration",
	}
	cmd.AddCommand(cliutil.NewHashCmd())
	return cmd
}

func resolveServerURL(flagOverride string) string {
	if url := strings.TrimSpace(flagOverride); url != "" {
		return url
	}
	if url := strings.TrimSpace(viper.GetString("server_url")); url != "" {
		return url
	}
	return arxconfig.CLIConfigFromViper().ServerURL
}

func newLoginCmd() *cobra.Command {
	var (
		serverURL string
		username  string
		password  string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with admin credentials and store a JWT locally",
		RunE: func(_ *cobra.Command, _ []string) error {
			return login.Run(login.Options{
				ServerURL: resolveServerURL(serverURL),
				Username:  username,
				Password:  password,
			})
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL from ~/.arx/cli.yaml")
	cmd.Flags().StringVarP(&username, "username", "", "", "Admin username (skips prompt)")
	cmd.Flags().StringVarP(&password, "password", "", "", "Admin password (skips prompt; use only in automation)")
	return cmd
}

func newUICmd() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive terminal UI",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := clicfg.Load()
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.Token) == "" {
				return fmt.Errorf("not logged in; run arx-ca-cli login first")
			}

			url := resolveServerURL(serverURL)
			if saved := strings.TrimSpace(cfg.ServerURL); saved != "" && strings.TrimSpace(serverURL) == "" {
				url = saved
			}

			client, err := cliapi.NewClient(url, cfg.BearerToken())
			if err != nil {
				return err
			}
			return tui.Run(client)
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL from ~/.arx/cli.yaml")
	return cmd
}
