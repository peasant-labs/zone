package main

import (
	"os"

	"github.com/peasant-labs/zone/cmd"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
