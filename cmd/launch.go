package cmd

import (
	"context"
	"errors"
	"fmt"
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

var launchCmd = &cobra.Command{
	Use:     "launch",
	Aliases: []string{"up"},
	Short:   "Build (if needed) and attach to the container",
	Long: `Build the Docker image (if needed) and attach to the container.

If no zone.toml exists and --harness is provided, creates a minimal
config automatically (zero-config quickstart). Reattaches to a running
container instead of creating a duplicate.`,
	Example: `  zone launch
  zone launch --headless -p "fix the tests"
  zone launch -P 3000:3000 -P 8080:8080
  zone launch -- --model sonnet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		plainFlag, _ := cmd.Root().PersistentFlags().GetBool("plain")
		harnessName, _ := cmd.Flags().GetString("harness")

		// Handle zero-config path: --harness provided, no zone.toml exists.
		if harnessName != "" {
			_, loadErr := config.LoadRepo(cwd + "/zone.toml")
			if loadErr != nil && (errors.Is(loadErr, config.ErrNoConfig) || os.IsNotExist(loadErr)) {
				// Generate a minimal zone.toml without requiring Docker.
				if err2 := docker.QuickstartWriteZoneToml(cwd, harnessName); err2 != nil {
					return fmt.Errorf("create zone.toml: %w", err2)
				}
			}
		} else {
			// No --harness flag: check if zone.toml exists.
			_, loadErr := config.LoadRepo(cwd + "/zone.toml")
			if loadErr != nil && (errors.Is(loadErr, config.ErrNoConfig) || os.IsNotExist(loadErr)) {
				// D-05: no zone.toml and no --harness — in TTY launch init wizard inline
				if tui.IsTTY(plainFlag) {
					detected := buildDetectionMap(cwd)
					wizard := tui.NewInitWizard(detected)
					finalModel, tuiErr := tui.RunTUI(wizard)
					if tuiErr != nil {
						return fmt.Errorf("init wizard: %w", tuiErr)
					}
					result := finalModel.(tui.InitWizard)
					if result.Cancelled {
						return fmt.Errorf("init cancelled")
					}
					if result.Err != nil {
						return result.Err
					}
					harnessName = result.SelectedHarness

					// Write zone.toml with selected harness
					tomlPath := filepath.Join(cwd, "zone.toml")
					content := generateInitTemplate(harnessName)
					if writeErr := os.WriteFile(tomlPath, []byte(content), 0644); writeErr != nil {
						return fmt.Errorf("write zone.toml: %w", writeErr)
					}
				} else {
					return fmt.Errorf("no zone.toml found. Run 'zone init --harness <name>' or 'zone launch --harness <name>'")
				}
			}
		}

		cfg, _, err := loadMergedFromDir(cwd)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// --harness flag overrides the harness in config.
		if harnessName != "" {
			cfg.Zone.Harness = harnessName
		}

		c := cache.New(cwd)
		if err := c.EnsureDir(); err != nil {
			return fmt.Errorf("ensure cache dir: %w", err)
		}
		if err := cache.EnsureGitignore(cwd); err != nil {
			return fmt.Errorf("update gitignore: %w", err)
		}

		mgr, err := docker.NewManager(cfg, c, cwd, version)
		if err != nil {
			return err
		}

		headless, _ := cmd.Flags().GetBool("headless")
		prompt, _ := cmd.Flags().GetString("prompt")
		rebuild, _ := cmd.Flags().GetBool("rebuild")
		noCache, _ := cmd.Flags().GetBool("no-cache")
		ports, _ := cmd.Flags().GetStringArray("port")

		opts := docker.LaunchOpts{
			Headless:    headless,
			Prompt:      prompt,
			Rebuild:     rebuild,
			NoCache:     noCache,
			HarnessArgs: args,
			Ports:       ports,
		}

		// TUI build progress: gate on TTY and non-headless mode.
		if tui.IsTTY(plainFlag) && !opts.Headless {
			needsBuild := mgr.NeedsBuild(ctx, opts.Rebuild)
			if needsBuild {
				linesCh := make(chan docker.BuildLine, 100)
				resultCh := make(chan docker.BuildResult, 1)
				mgr.BuildWithProgress(ctx, opts.NoCache, linesCh, resultCh)

				model := tui.NewBuildProgress(linesCh, resultCh, cancel)
				final, tuiErr := tui.RunTUI(model)
				if tuiErr != nil {
					return fmt.Errorf("build progress: %w", tuiErr)
				}
				bp := final.(tui.BuildProgress)
				if bp.BuildErr != nil {
					return bp.BuildErr
				}
				// Build done, image cached. Launch will skip build.
			}
		}

		return mgr.Launch(ctx, opts)
	},
}

func init() {
	launchCmd.Flags().String("harness", "", "Override harness name (enables zero-config)")
	launchCmd.Flags().Bool("headless", false, "Detached mode: print container ID and return")
	launchCmd.Flags().StringP("prompt", "p", "", "Prompt to pass to the harness")
	launchCmd.Flags().Bool("rebuild", false, "Force rebuild before launch")
	launchCmd.Flags().Bool("no-cache", false, "Build without Docker cache")
	launchCmd.Flags().StringArrayP("port", "P", nil, "Ad-hoc port binding (e.g., -P 3000:3000), repeatable")
	launchCmd.Flags().Bool("root", false, "Reserved for future use")
	launchCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts (reserved)")
}
