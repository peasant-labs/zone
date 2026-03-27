package main

import (
	"errors"
	"os"

	"github.com/peasant-labs/zone/cmd"
	"github.com/peasant-labs/zone/internal/cache"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, cache.ErrLockContention) {
			os.Exit(5)
		}
		os.Exit(1)
	}
}
