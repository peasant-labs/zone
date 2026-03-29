package cmd

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove the .zone/ cache directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		c := cache.New(cwd)

		// Check if lock is held; warn but proceed (per CONTEXT.md decision)
		if pid := cache.ReadLockPID(c.Dir()); pid > 0 {
			fmt.Fprintf(os.Stderr,
				"Warning: another zone process (PID %d) may be running. Cleaning anyway.\n", pid)
		}

		if err := c.Clean(); err != nil {
			return fmt.Errorf("clean cache: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Removed .zone/ cache directory.")

		removeImage, _ := cmd.Flags().GetBool("image")
		if removeImage {
			cfg, _, cfgErr := config.LoadMerged(cwd)
			if cfgErr != nil {
				return fmt.Errorf("load config for --image: %w", cfgErr)
			}
			// Re-create a fresh cache (dir was just cleaned).
			freshCache := cache.New(cwd)
			mgr, mgrErr := docker.NewManager(cfg, freshCache, cwd, version)
			if mgrErr != nil {
				return mgrErr
			}
			if err2 := mgr.RemoveImage(cmd.Context()); err2 != nil {
				return fmt.Errorf("remove image: %w", err2)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Removed Docker image.")
		}

		return nil
	},
}

func init() {
	cleanCmd.Flags().Bool("image", false, "Also remove the cached Docker image")
}
