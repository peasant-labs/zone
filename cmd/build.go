package cmd

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Force-rebuild the Docker image without launching",
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

		noCache, _ := cmd.Flags().GetBool("no-cache")
		if _, err := mgr.Build(cmd.Context(), noCache); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Image built successfully.")
		return nil
	},
}

func init() {
	buildCmd.Flags().Bool("no-cache", false, "Build without Docker cache")
}
