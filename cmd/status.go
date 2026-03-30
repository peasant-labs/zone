package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Show container state, harness, uptime, and resources",
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

		info, err := mgr.Status(ctx)
		if err != nil {
			return err
		}

		jsonMode, _ := cmd.Flags().GetBool("json")
		if jsonMode {
			b, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal JSON: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Repo:       %s\n", cwd)
		fmt.Fprintf(w, "Harness:    %s\n", cfg.Zone.Harness)
		fmt.Fprintf(w, "Container:  %s\n", strings.TrimPrefix(info.Name, "/"))

		state := info.State.Status
		if state == "running" && info.State.StartedAt != "" {
			started, err := time.Parse(time.RFC3339Nano, info.State.StartedAt)
			if err == nil {
				state = fmt.Sprintf("running (up %s)", formatDuration(time.Since(started)))
			}
		}
		fmt.Fprintf(w, "Status:     %s\n", state)
		fmt.Fprintf(w, "Image:      %s\n", info.Image)

		if info.NetworkSettings != nil {
			networks := make([]string, 0, len(info.NetworkSettings.Networks))
			for name := range info.NetworkSettings.Networks {
				networks = append(networks, name)
			}
			sort.Strings(networks)
			if len(networks) > 0 {
				fmt.Fprintf(w, "Network:    %s\n", strings.Join(networks, ", "))
			}
		}

		if info.HostConfig != nil && len(info.HostConfig.PortBindings) > 0 {
			ports := make([]string, 0)
			for containerPort, bindings := range info.HostConfig.PortBindings {
				for _, b := range bindings {
					ports = append(ports, fmt.Sprintf("%s:%s->%s", b.HostIP, b.HostPort, containerPort))
				}
			}
			sort.Strings(ports)
			fmt.Fprintf(w, "Ports:      %s\n", strings.Join(ports, ", "))
		}

		if info.HostConfig != nil {
			if info.HostConfig.Memory > 0 {
				fmt.Fprintf(w, "Memory:     %dMB\n", info.HostConfig.Memory/1024/1024)
			}
			if info.HostConfig.NanoCPUs > 0 {
				fmt.Fprintf(w, "CPUs:       %.1f\n", float64(info.HostConfig.NanoCPUs)/1e9)
			}
			if info.HostConfig.PidsLimit != nil && *info.HostConfig.PidsLimit > 0 {
				fmt.Fprintf(w, "PID Limit:  %d\n", *info.HostConfig.PidsLimit)
			}
		}

		if len(info.Mounts) > 0 {
			fmt.Fprintln(w, "Mounts:")
			for _, m := range info.Mounts {
				mode := "ro"
				if m.RW {
					mode = "rw"
				}
				fmt.Fprintf(w, "  %s -> %s (%s)\n", m.Source, m.Destination, mode)
			}
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().Bool("json", false, "Output raw container inspection as JSON")
}
