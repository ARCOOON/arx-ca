package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	agentapi "github.com/your-org/arx-ca/internal/agent/api"
	"github.com/your-org/arx-ca/internal/models"
)

// ListOptions configures a public certificate listing from arx-ca-server.
type ListOptions struct {
	APIURL string
}

// List prints the read-only certificate catalog from the server.
func List(ctx context.Context, opts ListOptions) error {
	client, err := agentapi.NewClient(opts.APIURL)
	if err != nil {
		return err
	}

	resp, err := client.ListPublicCertificates(ctx)
	if err != nil {
		return fmt.Errorf("list public certificates: %w", err)
	}

	if resp.Total == 0 {
		fmt.Println("No public certificates are published by the server.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERIAL\tSUBJECT\tNOT AFTER\tREVOKED")
	for _, cert := range resp.Certificates {
		revoked := "no"
		if cert.Revoked {
			revoked = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cert.Serial, cert.Subject, cert.NotAfter, revoked)
	}
	return w.Flush()
}

// DownloadOptions configures downloading a public certificate PEM from the server.
type DownloadOptions struct {
	APIURL string
	Serial string
	Output string
	Kind   string
}

// Download fetches a public leaf or intermediate certificate PEM and writes it to disk.
// Only certificate PEM is ever written; private keys are never requested or stored.
func Download(ctx context.Context, opts DownloadOptions) error {
	client, err := agentapi.NewClient(opts.APIURL)
	if err != nil {
		return err
	}

	kind := strings.ToLower(strings.TrimSpace(opts.Kind))
	var pem string
	switch kind {
	case "", "leaf", "certificate":
		if strings.TrimSpace(opts.Serial) == "" {
			return fmt.Errorf("serial is required for leaf certificate downloads")
		}
		pem, err = client.DownloadCertificatePEM(ctx, opts.Serial)
		if err != nil {
			return fmt.Errorf("download certificate: %w", err)
		}
	case "intermediate":
		pem, err = client.FetchIntermediatePEM(ctx)
		if err != nil {
			return fmt.Errorf("download intermediate CA: %w", err)
		}
	case "root":
		pem, err = client.FetchRootPEM(ctx)
		if err != nil {
			return fmt.Errorf("download root CA: %w", err)
		}
	default:
		return fmt.Errorf("unknown certificate kind %q (use leaf, intermediate, or root)", kind)
	}

	outPath := strings.TrimSpace(opts.Output)
	if outPath == "" {
		outPath = defaultOutputPath(opts.Serial, kind)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil && filepath.Dir(outPath) != "." {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(pem), 0o644); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}

	fmt.Printf("Public certificate PEM saved to %s\n", outPath)
	return nil
}

func defaultOutputPath(serial, kind string) string {
	name := strings.TrimSpace(serial)
	if name == "" {
		name = kind
	}
	if name == "" {
		name = "certificate"
	}
	return name + ".pem"
}

// FormatCertificateDetails renders a single public certificate summary.
func FormatCertificateDetails(cert models.PublicCertificateSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Serial:     %s\n", cert.Serial)
	fmt.Fprintf(&b, "Subject:    %s\n", cert.Subject)
	fmt.Fprintf(&b, "Not Before: %s\n", cert.NotBefore)
	fmt.Fprintf(&b, "Not After:  %s\n", cert.NotAfter)
	fmt.Fprintf(&b, "Revoked:    %v\n", cert.Revoked)
	if len(cert.DNSNames) > 0 {
		fmt.Fprintf(&b, "DNS Names:  %s\n", strings.Join(cert.DNSNames, ", "))
	}
	if len(cert.IPAddresses) > 0 {
		fmt.Fprintf(&b, "IP SANs:    %s\n", strings.Join(cert.IPAddresses, ", "))
	}
	return b.String()
}
