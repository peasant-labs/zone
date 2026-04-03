package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/peasant-labs/zone/internal/network"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove the .zone/ cache directory",
	Long: `Remove the .zone/ cache directory for this repo.

Removes cached config hashes, container IDs, and build logs.
Use --image to also remove the Docker image.`,
	Example: `  zone clean
  zone clean --image`,
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

		// Attempt firewall cleanup before deleting cache (D-39).
		// Uses the container hash derived from the repo path and cleans
		// tagged iptables rules. Best-effort: if sudo/iptables unavailable
		// or not on Linux, this is silently skipped.
		if runtime.GOOS == "linux" {
			containerName := docker.ContainerName(cwd)
			if len(containerName) >= 16 {
				containerHash := containerName[len(containerName)-16:]
				ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				cleanErr := network.RemoveRulesByHash(ctx, nil, containerHash)
				cancel()
				if cleanErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove firewall rules: %v\n", cleanErr)
				}
			}
		}

		if err := c.Clean(); err != nil {
			return fmt.Errorf("clean cache: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Removed .zone/ cache directory.")

		removeImage, _ := cmd.Flags().GetBool("image")
		if removeImage {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

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
			if err2 := mgr.RemoveImage(ctx); err2 != nil {
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
