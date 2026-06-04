package arxcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ARCOOON/arx-ca/internal/ui"
)

func newServerUICmd() *cobra.Command {
	uiRoot := &cobra.Command{
		Use:   "ui",
		Short: "Manage the dedicated WebUI static server",
		Long:  "Download release-matching WebUI assets from GitHub and enable the webui block in server.yaml.",
	}

	download := &cobra.Command{
		Use:   "download",
		Short: "Download and install WebUI assets from GitHub",
		Long: `Detects the running arx binary version, downloads webui-dist.tar.gz from the
matching ARCOOON/arx-ca GitHub release (or latest when built as v0.0.0-dev), extracts
static assets into webui.ui_dir, and sets webui.enabled to true in server.yaml.

Requires outbound HTTPS access to api.github.com and github.com. The target ui_dir
must be writable by the current user (use sudo when installing under /opt/arx).`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := ui.DownloadAndBootstrapWebUI(serverConfigFlag); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				return err
			}
			return nil
		},
	}

	uiRoot.AddCommand(download)
	return uiRoot
}
