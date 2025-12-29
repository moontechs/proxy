package main

import (
	"os"

	"github.com/moontechs/proxy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
