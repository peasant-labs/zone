package cmd

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:     "stop",
	Aliases: []string{"down"},
	Short:   "Stop and remove the container and network",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		cfg, _, err := config.LoadMerged(cwd)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		c := cache.New(cwd)
		mgr, err := docker.NewManager(cfg, c, cwd, version)
		if err != nil {
			return err
		}

		return mgr.Stop(cmd.Context())
	},
}

func init() {
	stopCmd.Flags().Int("timeout", 10, "Seconds to wait for container to stop gracefully")
}
