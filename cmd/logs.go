package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/peasant-labs/zone/internal/tui"
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
		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetString("tail")
		jsonMode, _ := cmd.Flags().GetBool("json")
		plainFlag, _ := cmd.Root().PersistentFlags().GetBool("plain")

		if tail == "" {
			tail = "all"
		}

		// --build mode: read the last build log file.
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

			// Show in TUI viewer when stdin and stdout are both terminals.
			if tui.IsTTY(plainFlag) && tui.IsOutputTTY() && !jsonMode {
				model := tui.NewLogViewer(string(data), nil, false)
				if _, err := tui.RunTUI(model); err != nil { // D-27: panic-safe
					return fmt.Errorf("log viewer: %w", err)
				}
				return nil
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		}

		// Container logs mode.
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

		// Non-TUI paths: --json, --plain, piped stdout, or non-TTY stdin.
		// Pitfall 3 / D-20: BOTH stdin and stdout must be terminals for TUI.
		if jsonMode || !tui.IsTTY(plainFlag) || !tui.IsOutputTTY() {
			opts := docker.LogsOpts{Follow: follow, Tail: tail, JSON: jsonMode}
			return mgr.Logs(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		}

		// TUI path — both stdin and stdout are terminals.
		if follow {
			// Fetch existing log snapshot first.
			var buf bytes.Buffer
			initialOpts := docker.LogsOpts{Follow: false, Tail: tail, JSON: false}
			if err := mgr.Logs(ctx, &buf, io.Discard, initialOpts); err != nil {
				return err
			}

			// Start a goroutine that tails new log lines and sends them on followCh.
			followCh := make(chan string, 100)
			go func() {
				defer close(followCh)
				pr, pw := io.Pipe()
				go func() {
					followOpts := docker.LogsOpts{Follow: true, Tail: "0", JSON: false}
					_ = mgr.Logs(ctx, pw, io.Discard, followOpts)
					pw.Close()
				}()
				scanner := bufio.NewScanner(pr)
				for scanner.Scan() {
					select {
					case followCh <- scanner.Text():
					case <-ctx.Done():
						return
					}
				}
			}()

			model := tui.NewLogViewer(buf.String(), followCh, true)
			if _, err := tui.RunTUI(model); err != nil { // D-27: panic-safe
				cancel() // stop follow goroutine
				return fmt.Errorf("log viewer: %w", err)
			}
			cancel() // stop follow goroutine
			return nil
		}

		// Non-follow TUI: fetch all logs and show in viewer.
		var buf bytes.Buffer
		opts := docker.LogsOpts{Follow: false, Tail: tail, JSON: false}
		if err := mgr.Logs(ctx, &buf, io.Discard, opts); err != nil {
			return err
		}

		model := tui.NewLogViewer(buf.String(), nil, false)
		if _, err := tui.RunTUI(model); err != nil { // D-27: panic-safe
			return fmt.Errorf("log viewer: %w", err)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (live tail)")
	logsCmd.Flags().Bool("build", false, "Show last Docker build log instead of container logs")
	logsCmd.Flags().String("tail", "", "Number of lines to show from end of logs (default: all)")
	logsCmd.Flags().Bool("json", false, "Output logs as JSON array with timestamps")
}
