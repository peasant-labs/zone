package cmd

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec -- <command>",
	Short: "Run a one-off command inside the running container",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no command specified. Usage: zone exec -- <command>")
		}

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

		asRoot, _ := cmd.Flags().GetBool("root")
		return mgr.Exec(cmd.Context(), args, asRoot)
	},
}

func init() {
	execCmd.Flags().Bool("root", false, "Run command as root inside the container")
}
