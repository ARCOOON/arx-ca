package arxcmd

import (
	"github.com/spf13/cobra"

	"github.com/your-org/arx-ca/internal/cli/util"
)

func utilHashCmd() *cobra.Command {
	return withCLIConfig(util.NewHashCmd())
}
