// log_viewer.go implements the log viewer with follow mode.
package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// logLineMsg carries a new log line from the follow channel into the tea loop.
type logLineMsg string

// LogViewer is a BubbleTea alt-screen model that renders log content in a
// scrollable viewport with follow mode and substring search.
//
// After RunTUI returns, check Err for any error that occurred.
type LogViewer struct {
	viewport    viewport.Model
	content     string   // full log content (joined for display)
	lines       []string // individual log lines for search
	followMode  bool     // auto-scroll to bottom when new lines arrive
	searchMode  bool     // user is typing a search query
	searchQuery string   // last executed search term
	searchInput string   // in-progress search input buffer
	matches     []int    // line indices that matched the last search
	matchIdx    int      // current match index for n/N navigation
	width       int
	height      int
	ready       bool        // true after first WindowSizeMsg initialises viewport
	logCh       <-chan string // non-nil in follow mode
	Err         error        // exported: any fatal error
}

// NewLogViewer creates a log viewer.
//
//   - initialContent is the full log text loaded so far.
//   - followCh is a channel that delivers new log lines in follow mode; nil if not following.
//   - startFollow indicates whether to start in follow mode (--follow flag).
func NewLogViewer(initialContent string, followCh <-chan string, startFollow bool) LogViewer {
	lines := strings.Split(initialContent, "\n")

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(22))
	vp.SoftWrap = true
	vp.SetContent(initialContent)

	m := LogViewer{
		viewport:   vp,
		content:    initialContent,
		lines:      lines,
		followMode: startFollow,
		logCh:      followCh,
		width:      80,
		height:     24,
	}

	if startFollow {
		vp.GotoBottom()
		m.viewport = vp
	}

	return m
}

// waitForLogLine returns a Cmd that blocks on the follow channel and delivers
// the next line as a logLineMsg.
func waitForLogLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil // channel closed — stop relaying
		}
		return logLineMsg(line)
	}
}

// Init implements tea.Model. Starts the channel relay if following.
func (m LogViewer) Init() tea.Cmd {
	if m.logCh != nil {
		return waitForLogLine(m.logCh)
	}
	return nil
}

// Update implements tea.Model.
func (m LogViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		viewportHeight := msg.Height - 2 // reserve 2 lines for status bar
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(viewportHeight)
		m.ready = true
		return m, nil

	case logLineMsg:
		// Append new line from follow channel
		line := string(msg)
		m.lines = append(m.lines, line)
		m.content = strings.Join(m.lines, "\n")
		m.viewport.SetContent(m.content)
		if m.followMode {
			m.viewport.GotoBottom()
		}
		// Re-apply search highlights if a search is active
		if m.searchQuery != "" {
			m.applySearchHighlights()
		}
		// Re-register to receive the next line
		return m, waitForLogLine(m.logCh)

	case tea.KeyPressMsg:
		if m.searchMode {
			return m.updateSearchMode(msg)
		}
		return m.updateNormalMode(msg)
	}

	// Pass all other messages (mouse scroll, etc.) to the viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateNormalMode handles key events when the search bar is not active.
func (m LogViewer) updateNormalMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "f":
		m.followMode = !m.followMode
		if m.followMode {
			m.viewport.GotoBottom()
		}
		return m, nil

	case "/":
		m.searchMode = true
		m.searchInput = ""
		return m, nil

	case "up":
		m.followMode = false // manual scroll disables follow
		m.viewport.ScrollUp(1)
		return m, nil

	case "down":
		m.viewport.ScrollDown(1)
		return m, nil

	case "n":
		if len(m.matches) > 0 {
			m.matchIdx = (m.matchIdx + 1) % len(m.matches)
			m.viewport.SetYOffset(m.matches[m.matchIdx])
			m.viewport.HighlightNext()
		}
		return m, nil

	case "N":
		if len(m.matches) > 0 {
			m.matchIdx = (m.matchIdx - 1 + len(m.matches)) % len(m.matches)
			m.viewport.SetYOffset(m.matches[m.matchIdx])
			m.viewport.HighlightPrevious()
		}
		return m, nil

	case "g":
		m.viewport.GotoTop()
		return m, nil

	case "G":
		m.viewport.GotoBottom()
		return m, nil

	default:
		// Let the viewport handle page-up/down and other built-in keys.
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
}

// updateSearchMode handles key events while the user is typing a search query.
func (m LogViewer) updateSearchMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput
		m.searchMode = false
		m.executeSearch()
		return m, nil

	case "escape":
		m.searchMode = false
		m.searchInput = ""
		return m, nil

	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		return m, nil

	default:
		// Append printable rune to input buffer.
		if r := msg.String(); len(r) == 1 {
			m.searchInput += r
		}
		return m, nil
	}
}

// executeSearch performs a case-insensitive substring search across all lines
// and updates the viewport highlights.
func (m *LogViewer) executeSearch() {
	m.matches = nil
	m.matchIdx = 0
	m.viewport.ClearHighlights()

	if m.searchQuery == "" {
		return
	}

	lowerQuery := strings.ToLower(m.searchQuery)
	for i, line := range m.lines {
		if strings.Contains(strings.ToLower(line), lowerQuery) {
			m.matches = append(m.matches, i)
		}
	}

	if len(m.matches) == 0 {
		return
	}

	// Jump viewport to first match
	m.viewport.SetYOffset(m.matches[0])

	// Build byte-offset match ranges across the full content for SetHighlights.
	m.applySearchHighlights()
}

// applySearchHighlights recomputes byte-offset ranges and applies them to the
// viewport for highlighting.
func (m *LogViewer) applySearchHighlights() {
	if m.searchQuery == "" {
		return
	}
	lowerContent := strings.ToLower(m.content)
	lowerQuery := strings.ToLower(m.searchQuery)

	var byteRanges [][]int
	offset := 0
	for {
		idx := strings.Index(lowerContent[offset:], lowerQuery)
		if idx < 0 {
			break
		}
		start := offset + idx
		end := start + len(lowerQuery)
		byteRanges = append(byteRanges, []int{start, end})
		offset = start + 1
		if offset >= len(lowerContent) {
			break
		}
	}

	if len(byteRanges) == 0 {
		return
	}

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("220")).
		Foreground(lipgloss.Color("0"))
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("202")).
		Foreground(lipgloss.Color("15"))

	m.viewport.HighlightStyle = highlightStyle
	m.viewport.SelectedHighlightStyle = selectedStyle
	m.viewport.SetHighlights(byteRanges)
}

// View implements tea.Model and returns an alt-screen view.
func (m LogViewer) View() tea.View {
	if !m.ready {
		v := tea.NewView("Initialising...")
		v.AltScreen = true
		return v
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("236")).
		Width(m.width)

	var statusLine string
	if m.searchMode {
		// Input prompt at the bottom
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("228")).
			Background(lipgloss.Color("236")).
			Width(m.width)
		promptLine := inputStyle.Render("Search: " + m.searchInput + "_")

		viewContent := m.viewport.View()
		content := viewContent + "\n" + promptLine

		view := tea.NewView(content)
		view.AltScreen = true
		return view
	}

	// Normal status bar
	followStatus := "follow: OFF"
	if m.followMode {
		followStatus = "follow: ON"
	}

	searchStatus := ""
	if m.searchQuery != "" {
		searchStatus = " | search: " + m.searchQuery
		if len(m.matches) > 0 {
			searchStatus += " (" + itoa(m.matchIdx+1) + "/" + itoa(len(m.matches)) + " matches)"
		} else {
			searchStatus += " (no matches)"
		}
	}

	help := "  q quit  / search  f follow  up/dn scroll  n/N next/prev  g/G top/btm"
	statusLine = statusStyle.Render(
		"  [" + followStatus + "]" + searchStatus + help,
	)

	content := m.viewport.View() + "\n" + statusLine

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// itoa converts an int to a decimal string without importing strconv/fmt at
// call sites.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
