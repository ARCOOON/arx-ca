package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	agentlocal "github.com/your-org/arx-ca/internal/agent/local"
	agentserver "github.com/your-org/arx-ca/internal/agent/server"
	agenttrust "github.com/your-org/arx-ca/internal/agent/trust"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arx-cert-service",
		Short: "Local certificate and trust store helper",
		Long:  "arx-cert-service inspects local certificate stores, manages trust anchors, and downloads public certificates from arx-ca-server. It never handles private keys from the server.",
	}

	root.AddCommand(newLocalCmd())
	root.AddCommand(newTrustCmd())
	root.AddCommand(newServerCmd())
	return root
}

func newLocalCmd() *cobra.Command {
	local := &cobra.Command{
		Use:   "local",
		Short: "List and view certificates installed on this machine",
	}

	var storeFilters []string
	list := &cobra.Command{
		Use:   "list",
		Short: "List certificates in system, user, and browser stores",
		RunE: func(_ *cobra.Command, _ []string) error {
			stores, err := agentlocal.ParseStoreKinds(storeFilters)
			if err != nil {
				return err
			}
			certs, err := agentlocal.List(agentlocal.ListOptions{Stores: stores})
			if err != nil {
				return err
			}
			if len(certs) == 0 {
				fmt.Println("No certificates found in the selected stores.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTORE\tLOCATION\tSUBJECT\tNOT AFTER")
			for _, cert := range certs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					cert.Thumbprint,
					cert.Store,
					cert.StoreName,
					cert.Subject,
					cert.NotAfter.Format("2006-01-02"),
				)
			}
			return w.Flush()
		},
	}
	list.Flags().StringSliceVar(&storeFilters, "store", nil, "Filter by store: system, user, browser")

	view := &cobra.Command{
		Use:   "view <id>",
		Short: "View details for an installed certificate by thumbprint or serial",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cert, err := agentlocal.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Thumbprint: %s\n", cert.Thumbprint)
			fmt.Printf("Store:      %s (%s)\n", cert.Store, cert.StoreName)
			fmt.Printf("Subject:    %s\n", cert.Subject)
			fmt.Printf("Issuer:     %s\n", cert.Issuer)
			fmt.Printf("Serial:     %s\n", cert.Serial)
			fmt.Printf("Valid:      %s — %s\n", cert.NotBefore.Format("2006-01-02"), cert.NotAfter.Format("2006-01-02"))
			fmt.Printf("Is CA:      %v\n", cert.IsCA)
			if len(cert.DNSNames) > 0 {
				fmt.Printf("DNS Names:  %s\n", joinComma(cert.DNSNames))
			}
			return nil
		},
	}

	local.AddCommand(list, view)
	return local
}

func newTrustCmd() *cobra.Command {
	trust := &cobra.Command{
		Use:   "trust",
		Short: "Install or remove Root and Intermediate CAs in local trust stores",
	}

	trust.AddCommand(newTrustInstallCmd("root", "Root", agenttrust.InstallRoot))
	trust.AddCommand(newTrustInstallCmd("intermediate", "Intermediate", agenttrust.InstallIntermediate))
	trust.AddCommand(newTrustUninstallCmd("root", "Root", agenttrust.UninstallRoot))
	trust.AddCommand(newTrustUninstallCmd("intermediate", "Intermediate", agenttrust.UninstallIntermediate))
	return trust
}

func newTrustInstallCmd(name, label string, install func(context.Context, string) error) *cobra.Command {
	var apiURL string
	cmd := &cobra.Command{
		Use:   "install-" + name,
		Short: fmt.Sprintf("Fetch and install the %s CA into local trust stores", label),
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if err := install(ctx, apiURL); err != nil {
				return err
			}
			fmt.Printf("%s CA installed into local trust stores.\n", label)
			fmt.Println("State saved under ~/.arx-cert-service/")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "url", "", "Base URL of arx-ca-server (e.g. http://localhost:8080)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newTrustUninstallCmd(name, label string, uninstall func() error) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-" + name,
		Short: fmt.Sprintf("Remove the %s CA from local trust stores", label),
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := uninstall(); err != nil {
				return err
			}
			fmt.Printf("%s CA removed from local trust stores.\n", label)
			return nil
		},
	}
}

func newServerCmd() *cobra.Command {
	server := &cobra.Command{
		Use:   "server",
		Short: "Read-only access to public certificates published by arx-ca-server",
	}

	var apiURL string
	list := &cobra.Command{
		Use:   "list",
		Short: "List public certificates available on the server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return agentserver.List(ctx, agentserver.ListOptions{APIURL: apiURL})
		},
	}
	list.Flags().StringVar(&apiURL, "url", "", "Base URL of arx-ca-server")
	_ = list.MarkFlagRequired("url")

	var (
		serial string
		output string
		kind   string
		dlURL  string
	)
	download := &cobra.Command{
		Use:   "download",
		Short: "Download a public certificate PEM (leaf, intermediate, or root)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return agentserver.Download(ctx, agentserver.DownloadOptions{
				APIURL: dlURL,
				Serial: serial,
				Output: output,
				Kind:   kind,
			})
		},
	}
	download.Flags().StringVar(&dlURL, "url", "", "Base URL of arx-ca-server")
	download.Flags().StringVar(&serial, "serial", "", "Certificate serial (required for leaf downloads)")
	download.Flags().StringVarP(&output, "output", "o", "", "Output PEM file path")
	download.Flags().StringVar(&kind, "kind", "leaf", "Certificate kind: leaf, intermediate, or root")
	_ = download.MarkFlagRequired("url")

	server.AddCommand(list, download)
	return server
}

func joinComma(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ", " + items[i]
	}
	return out
}
