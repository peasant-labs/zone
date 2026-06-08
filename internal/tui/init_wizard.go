// Package tui provides BubbleTea terminal UI components for zone.
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// screen represents which screen is active in the wizard.
type screen int

const (
	screenSelector screen = iota
	screenPreview
)

// harnessItem implements list.DefaultItem for the harness selector.
type harnessItem struct {
	name     string
	desc     string
	detected bool
}

func (h harnessItem) Title() string { return h.name }
func (h harnessItem) Description() string {
	if h.detected {
		return h.desc + "  * detected"
	}
	return h.desc
}
func (h harnessItem) FilterValue() string { return h.name }

// InitWizard is a two-screen BubbleTea model for the zone init wizard.
// After Run() completes, callers should read SelectedHarness, Confirmed,
// Cancelled, and Err to determine the outcome.
type InitWizard struct {
	// Exported result fields — read by cmd/init.go after RunTUI returns.
	SelectedHarness string
	Confirmed       bool
	Cancelled       bool
	Err             error

	// internal state
	screen          screen
	list            list.Model
	skipPermissions bool
	statusMessage   string
	width           int
	height          int
}

// NewInitWizard creates a wizard. detectedHarnesses is a map of harness
// name -> true for harnesses that have indicator files in the directory.
func NewInitWizard(detectedHarnesses map[string]bool) InitWizard {
	items := []list.Item{
		harnessItem{"claude-code", "Anthropic Claude Code", detectedHarnesses["claude-code"]},
		harnessItem{"opencode", "OpenCode", detectedHarnesses["opencode"]},
		harnessItem{"aider", "Aider (stub)", detectedHarnesses["aider"]},
		harnessItem{"gemini-cli", "Google Gemini CLI (stub)", detectedHarnesses["gemini-cli"]},
		harnessItem{"codex-cli", "OpenAI Codex CLI", detectedHarnesses["codex-cli"]},
		harnessItem{"custom", "Custom harness", false},
	}

	const defaultWidth = 80
	const defaultHeight = 14

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	l.Title = "Select your LLM harness"
	// Disable the built-in quit key so we can handle q ourselves
	l.DisableQuitKeybindings()

	return InitWizard{
		list:   l,
		width:  defaultWidth,
		height: defaultHeight,
	}
}

// Init implements tea.Model.
func (m InitWizard) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m InitWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyPressMsg:
		switch m.screen {
		case screenSelector:
			return m.updateSelector(msg)
		case screenPreview:
			return m.updatePreview(msg)
		}
	}

	if m.screen == screenSelector {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m InitWizard) updateSelector(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.Cancelled = true
		return m, tea.Quit
	case "enter":
		selected := m.list.SelectedItem()
		if selected == nil {
			return m, nil
		}
		item, ok := selected.(harnessItem)
		if !ok {
			return m, nil
		}
		m.SelectedHarness = item.name
		m.screen = screenPreview
		m.statusMessage = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m InitWizard) updatePreview(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.Cancelled = true
		return m, tea.Quit
	case "enter":
		m.Confirmed = true
		return m, tea.Quit
	case "s":
		m.skipPermissions = !m.skipPermissions
		m.statusMessage = ""
		return m, nil
	case "n":
		m.statusMessage = "[n] network sandboxing: coming soon (Phase 10)"
		return m, nil
	case "c":
		m.statusMessage = "[c] customize: coming soon"
		return m, nil
	}
	return m, nil
}

// View implements tea.Model.
func (m InitWizard) View() tea.View {
	var content string
	switch m.screen {
	case screenSelector:
		content = m.list.View()
	case screenPreview:
		content = m.renderPreview()
	}
	v := tea.NewView(content)
	v.WindowTitle = "zone init"
	return v
}

// renderPreview builds the config preview screen.
func (m InitWizard) renderPreview() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#AD58B4")).
		Padding(1, 2).
		Width(60)

	skipStr := "false"
	if m.skipPermissions {
		skipStr = "true"
	}

	details := strings.Join([]string{
		fmt.Sprintf("  Harness:          %s", m.SelectedHarness),
		fmt.Sprintf("  Base image:       ubuntu:24.04"),
		fmt.Sprintf("  Packages:         git, curl, ripgrep"),
		fmt.Sprintf("  Network mode:     none"),
		fmt.Sprintf("  Auth:             mount_home_config: true"),
		fmt.Sprintf("  skip_permissions: %s", skipStr),
	}, "\n")

	box := boxStyle.Render(details)

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	hotkeys := helpStyle.Render(
		"  [s] toggle skip_permissions  [n] network sandboxing  [c] customize\n" +
			"  [Enter] confirm and write    [q] cancel",
	)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  Configuration preview")
	lines = append(lines, "")
	lines = append(lines, box)
	lines = append(lines, "")
	lines = append(lines, hotkeys)
	if m.statusMessage != "" {
		lines = append(lines, "")
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6FF8"))
		lines = append(lines, "  "+statusStyle.Render(m.statusMessage))
	}
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}
