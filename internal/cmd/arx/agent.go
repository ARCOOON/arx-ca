package arxcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	agentenroll "github.com/your-org/arx-ca/internal/agent/enroll"
	agentlocal "github.com/your-org/arx-ca/internal/agent/local"
	agentserver "github.com/your-org/arx-ca/internal/agent/server"
	agenttrust "github.com/your-org/arx-ca/internal/agent/trust"
	"github.com/your-org/arx-ca/internal/cli/runtime"
)

func newAgentCmd() *cobra.Command {
	agent := &cobra.Command{
		Use:   "agent",
		Short: "Local certificate stores, trust anchors, and public certificate access",
		Long:  "Inspect local certificate stores, install trust anchors, and download public certificates from the CA. Never handles private keys from the server.",
	}

	agent.AddCommand(
		newAgentEnrollCmd(),
		newAgentLocalCmd(),
		newAgentTrustCmd(),
		newAgentCertCmd(),
	)

	return withCLIConfig(agent)
}

func newAgentEnrollCmd() *cobra.Command {
	var (
		serverURL string
		domain    string
		ttl       string
	)

	cmd := &cobra.Command{
		Use:   "enroll",
		Short: "Request and store a certificate for a domain using admin credentials",
		Long:  "Calls POST /api/v1/certificates/auto with the stored admin JWT and saves the certificate and private key under ~/.arx-cert-service/enrolled/<domain>/.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(domain) == "" {
				return fmt.Errorf("--domain is required")
			}
			client, err := runtime.NewAuthenticatedClient(serverURL)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			meta, err := agentenroll.Run(ctx, client.AutoCertificate, agentenroll.Options{
				Domain: domain,
				TTL:    ttl,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Enrolled certificate for %s (serial %s).\n", meta.Domain, meta.Serial)
			fmt.Println("Files saved under ~/.arx-cert-service/enrolled/")
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "DNS name for the issued certificate (required)")
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL")
	cmd.Flags().StringVar(&ttl, "ttl", "", "Certificate lifetime (e.g. 24h, 720h)")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func resolveAgentURL(flagOverride string) (string, error) {
	return runtime.ResolveServerURL(runtime.URLResolveOptions{
		FlagOverride:  flagOverride,
		PersistFlag:   flagOverride != "",
		UseAgentState: true,
	})
}

func newAgentLocalCmd() *cobra.Command {
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

func newAgentTrustCmd() *cobra.Command {
	trust := &cobra.Command{
		Use:   "trust",
		Short: "Install or remove Root and Intermediate CAs in local trust stores",
	}

	trust.AddCommand(
		newAgentTrustInstallCmd("root", "Root", agenttrust.InstallRoot),
		newAgentTrustInstallCmd("intermediate", "Intermediate", agenttrust.InstallIntermediate),
		newAgentTrustUninstallCmd("root", "Root", agenttrust.UninstallRoot),
		newAgentTrustUninstallCmd("intermediate", "Intermediate", agenttrust.UninstallIntermediate),
	)
	return trust
}

func newAgentTrustInstallCmd(name, label string, install func(context.Context, string) error) *cobra.Command {
	var apiURL string
	cmd := &cobra.Command{
		Use:   "install-" + name,
		Short: fmt.Sprintf("Fetch and install the %s CA into local trust stores", label),
		RunE: func(cmd *cobra.Command, _ []string) error {
			url, err := resolveAgentURL(apiURL)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if err := install(ctx, url); err != nil {
				return err
			}
			fmt.Printf("%s CA installed into local trust stores.\n", label)
			fmt.Println("State saved under ~/.arx-cert-service/")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "url", "", "Base URL of the CA server (saved to config when set)")
	return cmd
}

func newAgentTrustUninstallCmd(name, label string, uninstall func() error) *cobra.Command {
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

func newAgentCertCmd() *cobra.Command {
	cert := &cobra.Command{
		Use:   "cert",
		Short: "Read-only access to public certificates published by the CA",
	}

	var apiURL string
	list := &cobra.Command{
		Use:   "list",
		Short: "List public certificates available on the server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url, err := resolveAgentURL(apiURL)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return agentserver.List(ctx, agentserver.ListOptions{APIURL: url})
		},
	}
	list.Flags().StringVar(&apiURL, "url", "", "Base URL of the CA server")

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
			url, err := resolveAgentURL(dlURL)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return agentserver.Download(ctx, agentserver.DownloadOptions{
				APIURL: url,
				Serial: serial,
				Output: output,
				Kind:   kind,
			})
		},
	}
	download.Flags().StringVar(&dlURL, "url", "", "Base URL of the CA server")
	download.Flags().StringVar(&serial, "serial", "", "Certificate serial (required for leaf downloads)")
	download.Flags().StringVarP(&output, "output", "o", "", "Output PEM file path")
	download.Flags().StringVar(&kind, "kind", "leaf", "Certificate kind: leaf, intermediate, or root")

	cert.AddCommand(list, download)
	return cert
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
