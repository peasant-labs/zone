package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	dockerclient "github.com/docker/docker/client"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all zone containers across all repos",
	Long: `List all zone-managed containers across all repositories.

Queries Docker for containers with the zone management label.
Does not require a zone.toml — works from any directory.`,
	Example: `  zone ls
  zone ls --running
  zone ls --json
  zone ls --quiet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("connect to Docker: %w", docker.ErrDockerNotRunning)
		}
		defer cli.Close()

		containers, err := docker.ListContainers(ctx, cli)
		if err != nil {
			return err
		}

		runningOnly, _ := cmd.Flags().GetBool("running")
		if runningOnly {
			filtered := make([]docker.ContainerInfo, 0, len(containers))
			for _, c := range containers {
				if c.State == "running" {
					filtered = append(filtered, c)
				}
			}
			containers = filtered
		}

		quietMode, _ := cmd.Flags().GetBool("quiet")
		jsonMode, _ := cmd.Flags().GetBool("json")

		if quietMode {
			for _, c := range containers {
				fmt.Fprintln(cmd.OutOrStdout(), c.Name)
			}
			return nil
		}

		if jsonMode {
			b, err := json.MarshalIndent(containers, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal JSON: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		if len(containers) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No zone containers found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tHARNESS\tSTATUS\tUPTIME\tREPO")
		for _, c := range containers {
			uptime := "-"
			if c.State == "running" {
				uptime = formatDuration(time.Since(c.StartedAt))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", c.Name, c.Harness, c.State, uptime, c.RepoPath)
		}
		return w.Flush()
	},
}

func init() {
	lsCmd.Flags().Bool("json", false, "Output as JSON array")
	lsCmd.Flags().Bool("running", false, "Show only running containers")
	lsCmd.Flags().Bool("quiet", false, "Print only container names")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, h)
}
