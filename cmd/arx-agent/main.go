package main

import (
	"fmt"
	"os"

	arxagentcmd "github.com/ARCOOON/arx-ca/internal/cmd/arxagent"
)

func main() {
	if err := arxagentcmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
