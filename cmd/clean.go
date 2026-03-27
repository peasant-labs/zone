package cmd

import (
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/cache"
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
		return nil
	},
}
