package main

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/cmd"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		msg, exitCode := cmd.MapError(err)
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		os.Exit(exitCode)
	}
}
