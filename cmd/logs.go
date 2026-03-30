package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"log"},
	Short:   "View harness output",
	Long: `View container logs or the last Docker build log.

Shows harness output from the running container. Use --follow for
live tailing, --tail to limit output, or --build for the build log.`,
	Example: `  zone logs
  zone logs --follow
  zone logs --tail 50
  zone logs --build`,
	RunE: func(cmd *cobra.Command, args []string) error {
		buildMode, _ := cmd.Flags().GetBool("build")
		if buildMode {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			logPath := filepath.Join(cwd, ".zone", "logs", "last_build.log")
			data, err := os.ReadFile(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no build log found. Run `zone build` first")
				}
				return fmt.Errorf("read build log: %w", err)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
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

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetString("tail")
		jsonMode, _ := cmd.Flags().GetBool("json")
		if tail == "" {
			tail = "all"
		}

		opts := docker.LogsOpts{Follow: follow, Tail: tail, JSON: jsonMode}
		return mgr.Logs(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (live tail)")
	logsCmd.Flags().Bool("build", false, "Show last Docker build log instead of container logs")
	logsCmd.Flags().String("tail", "", "Number of lines to show from end of logs (default: all)")
	logsCmd.Flags().Bool("json", false, "Output logs as JSON array with timestamps")
}
