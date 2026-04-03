// status_view.go implements the live container status display.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/docker/docker/api/types/container"

	dockerpkg "github.com/peasant-labs/zone/internal/docker"
	"github.com/peasant-labs/zone/internal/config"
)

// tickMsg is sent on the 2-second polling interval.
type tickMsg time.Time

// statusUpdateMsg carries the result of a Docker status fetch.
type statusUpdateMsg struct {
	info *container.InspectResponse
	err  error
}

// StatusView is a BubbleTea model that shows live container status in alt-screen.
// After RunTUI returns, callers read Action and Err.
type StatusView struct {
	ctx     context.Context
	mgr     *dockerpkg.Manager
	cfg     *config.MergedConfig
	cwd     string
	info    *container.InspectResponse
	err     error
	width   int
	height  int
	quitting bool

	// Exported result fields — read by cmd/status.go after RunTUI returns.
	Action string // "restart" or "stop" if user pressed r/s
	Err    error  // any error from polling
}

// NewStatusView creates a StatusView model ready to run.
func NewStatusView(ctx context.Context, mgr *dockerpkg.Manager, cfg *config.MergedConfig, cwd string) StatusView {
	return StatusView{
		ctx:    ctx,
		mgr:    mgr,
		cfg:    cfg,
		cwd:    cwd,
		width:  80,
		height: 24,
	}
}

// fetchStatus is a tea.Cmd that calls mgr.Status and returns a statusUpdateMsg.
func fetchStatus(ctx context.Context, mgr *dockerpkg.Manager) tea.Cmd {
	return func() tea.Msg {
		info, err := mgr.Status(ctx)
		return statusUpdateMsg{info: info, err: err}
	}
}

// Init implements tea.Model. Fetches initial status immediately.
func (m StatusView) Init() tea.Cmd {
	return fetchStatus(m.ctx, m.mgr)
}

// Update implements tea.Model.
func (m StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusUpdateMsg:
		m.info = msg.info
		m.err = msg.err
		// Schedule next tick — poll sequentially (Pitfall 7: not overlapping)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tickMsg:
		// Tick fires: fetch new status
		return m, fetchStatus(m.ctx, m.mgr)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.Action = "restart"
			return m, tea.Quit
		case "s":
			m.Action = "stop"
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View implements tea.Model. Returns alt-screen view per D-11.
func (m StatusView) View() tea.View {
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().Bold(true).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	var lines []string

	if m.err != nil {
		lines = append(lines, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.info == nil {
		lines = append(lines, "Loading...")
	} else {
		row := func(label, value string) string {
			return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
		}

		lines = append(lines, row("Repo", m.cwd))
		lines = append(lines, row("Harness", m.cfg.Zone.Harness))
		lines = append(lines, row("Container", strings.TrimPrefix(m.info.Name, "/")))

		// Status with uptime
		state := m.info.State.Status
		if state == "running" && m.info.State.StartedAt != "" {
			started, err := time.Parse(time.RFC3339Nano, m.info.State.StartedAt)
			if err == nil {
				state = fmt.Sprintf("running (up %s)", formatStatusDuration(time.Since(started)))
			}
		}
		lines = append(lines, row("Status", state))
		lines = append(lines, row("Image", m.info.Image))

		// Network
		if m.info.NetworkSettings != nil {
			networks := make([]string, 0, len(m.info.NetworkSettings.Networks))
			for name := range m.info.NetworkSettings.Networks {
				networks = append(networks, name)
			}
			sort.Strings(networks)
			if len(networks) > 0 {
				lines = append(lines, row("Network", strings.Join(networks, ", ")))
			}
		}

		// Ports
		if m.info.HostConfig != nil && len(m.info.HostConfig.PortBindings) > 0 {
			ports := make([]string, 0)
			for containerPort, bindings := range m.info.HostConfig.PortBindings {
				for _, b := range bindings {
					ports = append(ports, fmt.Sprintf("%s:%s->%s", b.HostIP, b.HostPort, containerPort))
				}
			}
			sort.Strings(ports)
			lines = append(lines, row("Ports", strings.Join(ports, ", ")))
		}

		// Resources
		if m.info.HostConfig != nil {
			if m.info.HostConfig.Memory > 0 {
				lines = append(lines, row("Memory", fmt.Sprintf("%dMB", m.info.HostConfig.Memory/1024/1024)))
			}
			if m.info.HostConfig.NanoCPUs > 0 {
				lines = append(lines, row("CPUs", fmt.Sprintf("%.1f", float64(m.info.HostConfig.NanoCPUs)/1e9)))
			}
			if m.info.HostConfig.PidsLimit != nil && *m.info.HostConfig.PidsLimit > 0 {
				lines = append(lines, row("PID Limit", fmt.Sprintf("%d", *m.info.HostConfig.PidsLimit)))
			}
		}

		// Mounts
		if len(m.info.Mounts) > 0 {
			lines = append(lines, labelStyle.Render("Mounts:"))
			for _, mount := range m.info.Mounts {
				mode := "ro"
				if mount.RW {
					mode = "rw"
				}
				lines = append(lines, fmt.Sprintf("  %s -> %s (%s)", mount.Source, mount.Destination, mode))
			}
		}
	}

	boxContent := strings.Join(lines, "\n")
	box := boxStyle.Render(boxContent)

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	footer := helpStyle.Render("  q quit  r restart  s stop")

	content := box + "\n\n" + footer

	v := tea.NewView(content)
	v.AltScreen = true // alt-screen per D-11
	return v
}

// formatStatusDuration formats a duration for display in status view.
// Duplicated from cmd/ls.go to avoid cross-package dependency.
func formatStatusDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
