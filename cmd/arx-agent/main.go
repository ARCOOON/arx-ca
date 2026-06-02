package main

import (
	"fmt"
	"os"

	arxagentcmd "github.com/your-org/arx-ca/internal/cmd/arxagent"
)

func main() {
	if err := arxagentcmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
