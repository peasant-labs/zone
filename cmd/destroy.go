package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Fully tear down container, image, network, and cache",
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

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Fprint(cmd.OutOrStdout(), "This will remove the container, image, volume, and all cached data. Continue? [y/N] ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}
		}

		return mgr.Destroy(cmd.Context())
	},
}

func init() {
	destroyCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
