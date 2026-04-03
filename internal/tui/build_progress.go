// build_progress.go implements the Docker build progress display.
package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/peasant-labs/zone/internal/docker"
)

// buildLineMsg carries a single line of Docker build output to the Update loop.
type buildLineMsg string

// buildResultMsg carries the final build outcome to the Update loop.
type buildResultMsg docker.BuildResult

// waitForBuildLine blocks on the lines channel and returns one line as a tea.Msg.
// Must be re-registered in Update after each buildLineMsg.
func waitForBuildLine(ch <-chan docker.BuildLine) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil // channel closed (result channel will fire next)
		}
		return buildLineMsg(line.Text)
	}
}

// waitForBuildResult blocks on the result channel and returns the final outcome.
func waitForBuildResult(ch <-chan docker.BuildResult) tea.Cmd {
	return func() tea.Msg {
		res, ok := <-ch
		if !ok {
			return nil
		}
		return buildResultMsg(res)
	}
}

// BuildProgress is a BubbleTea model that displays Docker build output inline
// (not alt-screen) with a spinner and a scrolling viewport.
//
// After RunTUI returns, callers read ImageID and BuildErr for the outcome.
type BuildProgress struct {
	// Exported result fields — read by cmd/launch.go after RunTUI returns.
	ImageID  string
	BuildErr error

	// internal state
	spinner  spinner.Model
	viewport viewport.Model
	lines    []string
	linesCh  <-chan docker.BuildLine
	resultCh <-chan docker.BuildResult
	cancelFn context.CancelFunc
	done     bool
	width    int
	height   int
}

const buildViewportHeight = 15

// NewBuildProgress creates a BuildProgress model ready to run.
// linesCh and resultCh must be the channels passed to docker.Manager.BuildWithProgress.
// cancelFn is called when the user presses Ctrl+C to cancel the build.
func NewBuildProgress(linesCh <-chan docker.BuildLine, resultCh <-chan docker.BuildResult, cancelFn context.CancelFunc) BuildProgress {
	s := spinner.New()
	s.Spinner = spinner.Dot

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(buildViewportHeight))

	return BuildProgress{
		spinner:  s,
		viewport: vp,
		linesCh:  linesCh,
		resultCh: resultCh,
		cancelFn: cancelFn,
		width:    80,
		height:   buildViewportHeight + 2,
	}
}

// Init implements tea.Model. Starts the spinner and waits for the first line
// and the result concurrently.
func (m BuildProgress) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForBuildLine(m.linesCh),
		waitForBuildResult(m.resultCh),
	)
}

// Update implements tea.Model.
func (m BuildProgress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case buildLineMsg:
		if string(msg) != "" {
			// Trim trailing newlines before storing so viewport looks clean
			line := strings.TrimRight(string(msg), "\n\r")
			if line != "" {
				m.lines = append(m.lines, line)
				m.viewport.SetContent(strings.Join(m.lines, "\n"))
				m.viewport.GotoBottom()
			}
		}
		// Re-register to wait for the next line
		return m, waitForBuildLine(m.linesCh)

	case buildResultMsg:
		m.done = true
		m.ImageID = msg.ImageID
		m.BuildErr = msg.Err
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			// Cancel the build context and quit
			if m.cancelFn != nil {
				m.cancelFn()
			}
			return m, tea.Quit
		case "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "up":
			m.viewport.ScrollUp(1)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		h := msg.Height - 2
		if h < 5 {
			h = 5
		}
		m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(h))
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
		m.viewport.GotoBottom()
		return m, nil
	}

	return m, nil
}

// View implements tea.Model. Returns inline (NOT alt-screen) view per D-09.
func (m BuildProgress) View() tea.View {
	stepCount := len(m.lines)

	var header string
	if m.done {
		if m.BuildErr != nil {
			header = fmt.Sprintf("Build failed: %v", m.BuildErr)
		} else {
			header = fmt.Sprintf("Build complete (%d steps)", stepCount)
		}
	} else {
		header = fmt.Sprintf("%s Building image...  (%d steps)", m.spinner.View(), stepCount)
	}

	content := header + "\n" + m.viewport.View()
	v := tea.NewView(content)
	// Inline rendering: v.AltScreen is NOT set (build progress is transient, per D-09)
	return v
}
