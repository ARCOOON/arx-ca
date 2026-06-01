package main

import (
	"fmt"
	"os"

	arxcmd "github.com/your-org/arx-ca/internal/cmd/arx"
)

func main() {
	if err := arxcmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
