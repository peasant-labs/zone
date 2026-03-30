package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and relaunch the container",
	Long: `Stop the running container and relaunch it.

Equivalent to 'zone stop' followed by 'zone launch'. Use --rebuild
to force a fresh image build during restart.`,
	Example: `  zone restart
  zone restart --rebuild`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

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

		// Stop the running container first.
		if err := mgr.Stop(ctx); err != nil {
			return err
		}

		rebuild, _ := cmd.Flags().GetBool("rebuild")

		opts := docker.LaunchOpts{
			Rebuild: rebuild,
		}

		return mgr.Launch(ctx, opts)
	},
}

func init() {
	restartCmd.Flags().Bool("rebuild", false, "Force rebuild before relaunch")
}
