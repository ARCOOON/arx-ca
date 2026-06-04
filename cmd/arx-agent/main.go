package main

import (
	"fmt"
	"os"

	arxagentcmd "github.com/ARCOOON/arx-ca/internal/cmd/arxagent"
	"github.com/ARCOOON/arx-ca/internal/version"
)

// Version and Commit are injected at link time via -ldflags.
var (
	Version = "v0.0.0-dev"
	Commit  = "unknown"
)

func main() {
	version.Version = Version
	if err := arxagentcmd.Execute(Version, Commit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
