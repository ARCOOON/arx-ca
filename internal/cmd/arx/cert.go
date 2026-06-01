package arxcmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/your-org/arx-ca/internal/cli/runtime"
)

func newCertCmd() *cobra.Command {
	cert := &cobra.Command{
		Use:   "cert",
		Short: "Manage certificates issued by the CA (authenticated admin API)",
	}

	cert.AddCommand(newCertListCmd(), newCertRevokeCmd())
	return withCLIConfig(cert)
}

func newCertListCmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List certificates issued by the CA",
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := runtime.NewAuthenticatedClient(serverURL)
			if err != nil {
				return err
			}
			list, err := client.ListCertificates(context.Background())
			if err != nil {
				return err
			}
			if len(list.Certificates) == 0 {
				fmt.Println("No issued certificates found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SERIAL\tSTATUS\tSUBJECT\tNOT AFTER")
			for _, c := range list.Certificates {
				status := "active"
				if c.Revoked {
					status = "revoked"
				}
				subject := c.Subject
				if len(subject) > 48 {
					subject = subject[:45] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					c.Serial,
					status,
					subject,
					c.NotAfter.Format("2006-01-02"),
				)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL")

	return cmd
}

func newCertRevokeCmd() *cobra.Command {
	var (
		serverURL string
		reason    string
	)

	cmd := &cobra.Command{
		Use:   "revoke <serial>",
		Short: "Revoke a certificate by serial number",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			client, err := runtime.NewAuthenticatedClient(serverURL)
			if err != nil {
				return err
			}
			resp, err := client.RevokeCertificate(context.Background(), args[0], reason)
			if err != nil {
				return err
			}
			fmt.Printf("Revoked %s at %s\n", resp.Serial, resp.RevokedAt)
			return nil
		},
	}
	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "Override the server URL")
	cmd.Flags().StringVar(&reason, "reason", "", "Revocation reason (informational; sent when the API supports it)")

	return cmd
}
