package arxcmd

import (
	"github.com/spf13/cobra"

	"github.com/ARCOOON/arx-ca/internal/ui"
)

var uiDownloadVersion string

func newServerUICmd() *cobra.Command {
	uiRoot := &cobra.Command{
		Use:   "ui",
		Short: "Manage the dedicated WebUI static server",
		Long:  "Download release-matching WebUI assets from GitHub and enable the webui block in server.yaml.",
	}

	download := &cobra.Command{
		Use:   "download",
		Short: "Download and install WebUI assets from GitHub",
		Long: `Downloads webui-dist.tar.gz from an ARCOOON/arx-ca GitHub release, extracts static
assets into webui.ui_dir, and sets webui.enabled to true in server.yaml.

By default the release tag matches the running arx binary version (or latest when built
as v0.0.0-dev). Use --version to fetch a specific release tag instead (for example when
downgrading or when the binary is a development build).

Requires outbound HTTPS access to api.github.com and github.com. The target ui_dir
must be writable by the current user (use sudo when installing under /opt/arx-ca).`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return ui.DownloadAndBootstrapWebUI(serverConfigFlag, uiDownloadVersion)
		},
	}
	download.Flags().StringVar(&uiDownloadVersion, "version", "", "GitHub release tag to download (for example v1.0.2); defaults to the arx binary version")

	uiRoot.AddCommand(download)
	return uiRoot
}
