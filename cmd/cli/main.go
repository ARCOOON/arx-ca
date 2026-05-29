package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cliapi "github.com/your-org/arx-ca/internal/cli/api"
	"github.com/your-org/arx-ca/internal/cli/config"
	"github.com/your-org/arx-ca/internal/cli/login"
	"github.com/your-org/arx-ca/internal/cli/tui"
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
	}

	root.AddCommand(newLoginCmd())
	root.AddCommand(newUICmd())
	return root
}

func newLoginCmd() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with admin credentials and store a JWT locally",
		RunE: func(_ *cobra.Command, _ []string) error {
			return login.Run(serverURL)
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Base URL of the arx-ca-server API (default http://localhost:8080)")
	return cmd
}

func newUICmd() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive terminal UI",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.Token) == "" {
				return fmt.Errorf("not logged in; run arx-ca-cli login first")
			}

			url := strings.TrimSpace(serverURL)
			if url == "" {
				url = strings.TrimSpace(cfg.ServerURL)
			}
			if url == "" {
				url = "http://localhost:8080"
			}

			client, err := cliapi.NewClient(url, cfg.BearerToken())
			if err != nil {
				return err
			}
			return tui.Run(client)
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL from config")
	return cmd
}
