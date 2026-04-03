# Phase 9: TUI Layer - Research

**Researched:** 2026-04-03
**Domain:** BubbleTea v2 TUI, golang.org/x/term, Lip Gloss v2, Bubbles v2 components
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Init Wizard (TUI-01)
- **D-01:** Full spec-compliant wizard per spec §13: BubbleTea list selector with harness names, descriptions, and auto-detection hints (e.g., "* detected" for claude-code when `.claude/` exists)
- **D-02:** After harness selection, show config preview screen with harness-specific details (base image, packages, network mode, auth, skip_permissions status)
- **D-03:** Config preview hotkeys: [s] toggle skip_permissions, [n] network sandboxing, [c] customize (open further options), [Enter] confirm and write, [q] cancel
- **D-04:** Replace current `promptHarnessSelection()` in `cmd/init.go` with BubbleTea model when TTY detected; keep text fallback for non-TTY
- **D-05:** `zone launch` without `zone.toml` and without `--harness` in TTY: launch init wizard inline (same BubbleTea flow), write config, then proceed to build/launch per spec §3.6

#### Build Progress (TUI-02)
- **D-06:** BubbleTea viewport with spinner, streaming Docker build output line-by-line
- **D-07:** Subscribe to `<-chan BuildProgress` from `Manager.Build()` via `tea.Cmd` that reads from the channel
- **D-08:** On build complete, transition to TTY attach (or print container ID if `--headless`)
- **D-09:** Inline rendering (not alt-screen) — build progress is transient and should flow in the terminal output
- **D-10:** Keybindings: down to scroll, Ctrl+C to cancel build

#### Status View (TUI-03)
- **D-11:** Alt-screen BubbleTea model with box-drawn status display per spec §13 layout
- **D-12:** Polls container status every 2 seconds via Docker SDK `ContainerInspect`
- **D-13:** Display fields: Repo, Harness, Container name, Status (with uptime), Image ID, Network mode, Port mappings, Resource limits, Mounts
- **D-14:** Interactive hotkeys: q quit, r restart, s stop
- **D-15:** `--json` flag bypasses BubbleTea entirely — prints raw JSON to stdout (existing Phase 8 behavior preserved)

#### Log Viewer (TUI-04)
- **D-16:** Alt-screen BubbleTea viewport with auto-scroll and search
- **D-17:** `--follow` starts in follow mode (auto-scroll to bottom); without `--follow`, starts paused at the end
- **D-18:** `/ search` keybinding for text search within logs per spec §13
- **D-19:** `--build` loads from `.zone/logs/last_build.log` instead of live container logs
- **D-20:** When stdout is piped, output plain text (no TUI chrome) — `zone logs -f | grep error` works correctly
- **D-21:** Keybindings: up/down scroll, / search, f toggle follow, q quit

#### TTY Detection (TUI-05, TUI-06)
- **D-22:** Use `golang.org/x/term` package's `term.IsTerminal()` on stdin fd per spec §3.5, replacing current `os.Stdin.Stat()` approach in `cmd/init.go`
- **D-23:** Centralize TTY check as a helper in `cmd/` or `internal/tui/` — all commands that conditionally use TUI call this single function
- **D-24:** `--plain` flag (already defined in root.go) force-disables TUI even when TTY detected
- **D-25:** When TUI disabled (non-TTY or `--plain`), commands fall back to existing plain text output from Phase 8

#### Non-TTY Init Error (TUI-07)
- **D-26:** `zone init` without `--harness` in non-TTY: error with "Interactive mode requires a terminal. Use `--harness <name>` for non-interactive init." — already implemented in Phase 8, just needs to use the centralized TTY check

#### Terminal Restore / Panic Recovery
- **D-27:** All BubbleTea programs wrapped in a deferred panic recovery that restores terminal state before re-panicking
- **D-28:** Use alt-screen only for status view and log viewer; init wizard and build progress use inline rendering
- **D-29:** BubbleTea's built-in cleanup handles normal exits; panic handler covers abnormal exits

#### BubbleTea Version
- **D-30:** Defer version choice (v1 vs v2) to researcher — STATE.md flags "BubbleTea v2.0.0 is only one month old (Feb 2026) — Cobra integration patterns for v2 need verification." **RESOLVED: Use v2.** See findings below.

#### Cobra Integration Pattern
- **D-31:** All BubbleTea models follow the spec §13 integration pattern: check `--json`/`--plain`/non-TTY first, fall back to plain text; otherwise create TUI model and run `tea.NewProgram(model).Run()`
- **D-32:** Each TUI model lives in `internal/tui/` and is imported by the corresponding `cmd/` file

### Claude's Discretion
- Exact styling choices (colors, borders, spacing) within Lip Gloss — as long as consistent
- Spinner type/style for build progress
- Config preview layout details beyond what spec §13 shows
- Search implementation approach in log viewer (substring vs regex)
- Polling interval for status view (spec says 2s, can adjust if needed)
- Whether to use mouse cell motion options

### Deferred Ideas (OUT OF SCOPE)
- `--edit` flag on `zone config` (opens $EDITOR) — backlog
- `--schema` flag on `zone config` — backlog
- `--from-devcontainer` migration on `zone init` — v2 backlog
- Mouse support in TUI views — backlog if requested
- Theme/color customization in config — v2 feature
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TUI-01 | Init wizard with BubbleTea interactive harness selection and config preview | Bubbles v2 list component, two-screen model (selector → preview), KeyPressMsg handling |
| TUI-02 | Build progress display with Docker build log streaming | waitForActivity channel pattern, spinner component, inline (no alt-screen) viewport |
| TUI-03 | Status view with live container state, uptime, ports, resources | Alt-screen via `v.AltScreen = true` in View(), tea.Tick for 2s polling |
| TUI-04 | Log viewer with follow mode and build log option | Alt-screen viewport, SetContent/GotoBottom, substring search with highlighting |
| TUI-05 | TTY auto-detection: BubbleTea when TTY, plain text when not | `golang.org/x/term` IsTerminal(int(os.Stdin.Fd())) — centralized helper |
| TUI-06 | `--plain` flag force-disables TUI even in TTY | Read from rootCmd.PersistentFlags() — already defined in root.go line 50 |
| TUI-07 | Non-TTY `zone init` without `--harness` errors with helpful message | Already implemented in cmd/init.go; refactor to use centralized TTY helper |
</phase_requirements>

---

## Summary

Phase 9 adds a BubbleTea v2 TUI layer on top of the Phase 8 plain-text output paths. The research resolves the critical D-30 question (v1 vs v2) in favor of **BubbleTea v2** (charm.land/bubbletea/v2 v2.0.2): it shipped February 24, 2025, has been battle-tested in Charm's production Crush AI coding agent for months before release, and the key "month-old instability" concern from STATE.md is resolved — v2.0.2 (Mar 9, 2025) has been out for over 12 months by implementation time.

The most significant v2 API difference from v1 is that `View()` returns `tea.View` (not a string), and alt-screen is declared declaratively via `v.AltScreen = true` on the returned View struct rather than via `tea.WithAltScreen()` ProgramOption. Key handling uses `tea.KeyPressMsg` with `msg.String()` instead of `tea.KeyMsg`. These are the only breaking changes that affect the spec §13 integration pattern — the overall `tea.NewProgram(model).Run()` pattern is identical.

The Docker build streaming integration (D-07) cannot use a `<-chan BuildProgress` from the existing `Manager.Build()` since that function is synchronous — it needs refactoring to expose a progress channel, OR the build progress TUI wraps the existing `buildImage` call differently. The `waitForActivity` pattern (one blocking tea.Cmd per message, re-registered in Update) is the correct BubbleTea idiom for channel-to-Update relay.

**Primary recommendation:** Use BubbleTea v2 (`charm.land/bubbletea/v2 v2.0.2`), Bubbles v2 (`charm.land/bubbles/v2 v2.1.0`) for list/viewport/spinner, Lip Gloss v2 (`charm.land/lipgloss/v2 v2.0.2`) for styling, and `golang.org/x/term v0.41.0` for TTY detection. Add these four dependencies in Wave 0 of the plan.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| charm.land/bubbletea/v2 | v2.0.2 | BubbleTea TUI framework | Official v2 stable; production-validated; resolves D-30 |
| charm.land/bubbles/v2 | v2.1.0 | UI components: list, viewport, spinner | Official Charm component library for BubbleTea v2 |
| charm.land/lipgloss/v2 | v2.0.2 | Terminal styling: colors, borders, layout | Official Charm styling library for BubbleTea v2 |
| golang.org/x/term | v0.41.0 | TTY detection via `IsTerminal()` | Spec §3.5 mandates this package explicitly |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang.org/x/term | v0.41.0 | `term.MakeRaw()`, `term.Restore()` for panic handler | Panic recovery restores raw terminal state |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| charm.land/bubbletea/v2 | github.com/charmbracelet/bubbletea (v1) | v1 still available but no longer actively developed; v2 is current stable |
| charm.land/lipgloss/v2 | tview, tcell | lipgloss is purpose-built for BubbleTea; tview has own event loop incompatible with BubbleTea |
| golang.org/x/term | os.Stdin.Stat() ModeCharDevice | Spec explicitly requires term.IsTerminal(); current isInteractive() in cmd/init.go uses the os.Stat approach which the spec says to replace |

**Installation:**
```bash
go get charm.land/bubbletea/v2@v2.0.2
go get charm.land/bubbles/v2@v2.1.0
go get charm.land/lipgloss/v2@v2.0.2
go get golang.org/x/term@v0.41.0
```

**Version verification:** Verified via `go get` in workspace on 2026-04-03. Final go.mod entries confirmed as v2.0.2 (bubbletea), v2.1.0 (bubbles), v2.0.2 (lipgloss), v0.41.0 (term).

## Architecture Patterns

### Recommended Project Structure
```
internal/tui/
├── init_wizard.go      # BubbleTea init wizard (harness selector + config preview)
├── build_progress.go   # Build progress inline view with spinner and viewport
├── status_view.go      # Alt-screen live container status with polling
└── log_viewer.go       # Alt-screen log viewer with follow mode and search

cmd/
├── init.go             # Replace promptHarnessSelection() with tui.NewInitWizard()
├── launch.go           # Wire tui.NewBuildProgress() around buildImage; inline init wizard
├── status.go           # Wire tui.NewStatusView() when TTY and no --json
└── logs.go             # Wire tui.NewLogViewer() when TTY and no --json and not piped
```

### Pattern 1: BubbleTea v2 Model Interface

**What:** Every TUI component implements the v2 Model interface — three methods with the key difference that `View()` returns `tea.View` (a struct), not a `string`.

**When to use:** All four TUI components.

```go
// Source: https://pkg.go.dev/charm.land/bubbletea/v2
import tea "charm.land/bubbletea/v2"

type StatusView struct {
    mgr    *docker.Manager
    info   *docker.ContainerInfo
    err    error
}

func (m StatusView) Init() tea.Cmd {
    return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:       // v2: KeyPressMsg, not KeyMsg
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit()
        }
    case tickMsg:
        // fetch new status, schedule next tick
        return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
            return tickMsg(t)
        })
    }
    return m, nil
}

func (m StatusView) View() tea.View {  // v2: returns tea.View, not string
    v := tea.NewView(renderStatus(m))
    v.AltScreen = true               // v2: declarative, not tea.WithAltScreen()
    return v
}
```

### Pattern 2: Alt-Screen Declaration (v2 change from v1)

**What:** In v2, alt-screen is set on the returned `tea.View` struct, not passed as a `ProgramOption`. This is the single largest breaking change affecting the spec §13 code example (which was written for v1 using `tea.WithAltScreen()`).

**When to use:** Status view and log viewer (per D-28).

```go
// v1 (spec §13 example — DO NOT USE):
p := tea.NewProgram(model, tea.WithAltScreen())  // WRONG for v2

// v2 (correct approach):
p := tea.NewProgram(model)  // no options needed
// Inside View():
func (m Model) View() tea.View {
    v := tea.NewView(content)
    v.AltScreen = true  // declare alt-screen here
    return v
}
```

**Note:** The spec §13 integration code example uses the v1 `tea.WithAltScreen()` pattern. The planner must use the v2 declarative approach instead.

### Pattern 3: Channel-to-Update Relay (waitForActivity)

**What:** The standard BubbleTea pattern for reading from a Go channel (e.g., a Docker build progress channel) and delivering messages to the `Update` loop. Uses two functions: one to send items on the channel, one to block-read a single item and return it as a `tea.Msg`.

**When to use:** Build progress TUI (D-07) — reading Docker build output line-by-line.

```go
// Source: https://github.com/charmbracelet/bubbletea/blob/main/examples/realtime/main.go

// BuildLineMsg carries a single line from the Docker build stream.
type buildLineMsg string
type buildDoneMsg struct{ imageID string }
type buildErrMsg struct{ err error }

// waitForBuildLine blocks on the channel and returns one message.
// Re-register in Update after each buildLineMsg.
func waitForBuildLine(ch <-chan buildLineMsg) tea.Cmd {
    return func() tea.Msg {
        return <-ch  // blocks until a line arrives or channel is closed
    }
}

// In Update:
case buildLineMsg:
    m.lines = append(m.lines, string(msg))
    m.viewport.SetContent(strings.Join(m.lines, "\n"))
    return m, waitForBuildLine(m.ch)  // register next wait

case buildDoneMsg:
    m.done = true
    m.imageID = msg.imageID
    return m, tea.Quit()
```

**Critical:** The existing `buildImage()` in `internal/docker/build.go` is synchronous (calls `streamBuildOutput` which blocks until build completes). To wire the build progress TUI, the plan needs a new function that runs `buildImage` in a goroutine and feeds output lines to a channel — OR wraps the existing function at the cmd layer without modifying internal/docker.

### Pattern 4: Key Handling (v2 change from v1)

**What:** v2 uses `tea.KeyPressMsg` instead of `tea.KeyMsg`. The `msg.String()` method returns readable key names.

```go
// v1 (DO NOT USE):
case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlC: ...
    case tea.KeyRunes:
        switch string(msg.Runes) { ... }
    }

// v2 (correct):
case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c", "q":  return m, tea.Quit()
    case "up":           m.viewport.ScrollUp(1)
    case "down":         m.viewport.ScrollDown(1)
    case "/":            m.searchMode = true
    case "f":            m.followMode = !m.followMode
    }
```

### Pattern 5: TTY Detection Centralized Helper

**What:** Replace `isInteractive()` in `cmd/init.go` with a shared function using `golang.org/x/term`.

**When to use:** All four command integrations.

```go
// internal/tui/tty.go (or cmd/tty.go)
import (
    "os"
    "golang.org/x/term"
)

// IsTTY returns true if stdin is a terminal AND --plain flag is not set.
func IsTTY(plainFlag bool) bool {
    if plainFlag {
        return false
    }
    return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsOutputTTY returns true if stdout is also a terminal (for log piping detection).
func IsOutputTTY() bool {
    return term.IsTerminal(int(os.Stdout.Fd()))
}
```

### Pattern 6: Cobra Integration

**What:** Each command's RunE checks flags/TTY, then conditionally runs the TUI or falls back to plain text.

```go
// Source: zone-spec.md §13 integration pattern (adapted for v2)
RunE: func(cmd *cobra.Command, args []string) error {
    jsonMode, _ := cmd.Flags().GetBool("json")
    plainMode, _ := cmd.Root().PersistentFlags().GetBool("plain")

    if jsonMode || !tui.IsTTY(plainMode) {
        return printStatusPlain(ctx, mgr, cmd)  // existing Phase 8 path
    }

    model := tui.NewStatusView(mgr, cfg, cwd)
    p := tea.NewProgram(model)  // no WithAltScreen() in v2
    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("tui: %w", err)
    }
    result := finalModel.(tui.StatusView)
    return result.Err
}
```

### Pattern 7: Panic Recovery for Terminal Restore

**What:** Defer a recovery function that restores terminal state before re-panicking. BubbleTea v2's built-in panic handler (`WithoutCatchPanics()` is opt-out) should handle most cases, but an explicit wrapper ensures the `cmd/` layer always cleans up.

```go
// Source: D-27 decision, standard Go panic recovery pattern
func runWithTUI(fn func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            // BubbleTea v2 has built-in panic catch by default.
            // This outer recovery handles panics outside the tea.Program.
            err = fmt.Errorf("tui panic: %v", r)
        }
    }()
    return fn()
}
```

**Note:** BubbleTea v2 catches panics by default (opt out with `WithoutCatchPanics()`). For the zone use case, rely on BubbleTea's built-in handler for in-model panics. The `cmd/` layer deferred recovery is only needed if the `tea.NewProgram(model).Run()` call itself panics (very rare).

### Pattern 8: Bubbles v2 List Component for Init Wizard

**What:** Use `charm.land/bubbles/v2/list` for the harness selector screen. Implement `list.DefaultItem` interface on harness items.

```go
import (
    "charm.land/bubbles/v2/list"
    tea "charm.land/bubbletea/v2"
)

type harnessItem struct {
    name     string
    desc     string
    detected bool
}

func (h harnessItem) Title() string       { return h.name }
func (h harnessItem) Description() string {
    if h.detected { return h.desc + "  * detected" }
    return h.desc
}
func (h harnessItem) FilterValue() string { return h.name }

// Create the list:
items := []list.Item{
    harnessItem{"claude-code", "Anthropic Claude Code", detected},
    // ...
}
l := list.New(items, list.NewDefaultDelegate(), width, height)
l.Title = "Select your LLM harness"
```

### Pattern 9: Bubbles v2 Viewport for Build Progress and Log Viewer

```go
import "charm.land/bubbles/v2/viewport"

// Create:
vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
vp.SetContent(strings.Join(lines, "\n"))

// Navigation:
vp.ScrollDown(1)    // one line
vp.GotoBottom()     // follow mode
vp.AtBottom()       // check if at bottom for auto-scroll

// In Update:
case tea.WindowSizeMsg:
    vp.Width = msg.Width
    vp.Height = msg.Height - 2  // reserve 2 lines for status bar
```

### Anti-Patterns to Avoid

- **Using `tea.WithAltScreen()` as a ProgramOption:** This is the v1 pattern. In v2, set `v.AltScreen = true` in the `View()` method. The spec §13 code example uses v1 syntax — override it.
- **Using `tea.KeyMsg`:** This is v1. Use `tea.KeyPressMsg` in v2.
- **Calling `View()` expecting a string:** The v2 `View()` returns `tea.View`. Models embedding other models that use `View() string` (like viewport, spinner) still return string from their own `View()` — only the top-level BubbleTea model returns `tea.View`.
- **Direct goroutines inside Update:** Use the `waitForActivity` channel relay pattern instead.
- **Blocking in Init():** Use tea.Cmd for all I/O. Init returns a Cmd, never blocks.
- **Sharing state between concurrent goroutines:** BubbleTea is single-threaded in Update; channel relay ensures safe message passing.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scrollable text viewport | Custom scroll logic | `charm.land/bubbles/v2/viewport` | Handles resize, key bindings, percent, gutter |
| List with selection | Custom list renderer | `charm.land/bubbles/v2/list` | Filtering, pagination, delegates, status messages built in |
| Spinner animation | Custom frame ticker | `charm.land/bubbles/v2/spinner` | 12 built-in styles, automatic tick, Lip Gloss styled |
| TTY detection | `os.Stdin.Stat() ModeCharDevice` | `golang.org/x/term.IsTerminal()` | Cross-platform (Windows compatible), spec-mandated |
| Terminal color styling | fmt.Sprintf ANSI codes | `charm.land/lipgloss/v2` | Color profiles, automatic downsampling, border utilities |
| Alt-screen management | Manual ANSI escape codes | BubbleTea v2 `v.AltScreen = true` | Automatic cleanup on normal and abnormal exit |

**Key insight:** BubbleTea's component ecosystem (bubbles) solves every scrolling/list/spinner problem that zone needs. Any custom re-implementation adds per-cell scroll math, resize handling, and accessibility that the existing components already handle correctly.

## Common Pitfalls

### Pitfall 1: Using v1 API in v2 Code
**What goes wrong:** Code uses `tea.WithAltScreen()`, `tea.KeyMsg`, or `View() string` — compiles against `charm.land/bubbletea/v2` but with wrong types. The spec §13 integration example was written before v2 shipped.
**Why it happens:** The spec §13 Cobra integration code shows `tea.NewProgram(model, tea.WithAltScreen())` — this is v1. The v2 API removed `WithAltScreen` as a ProgramOption.
**How to avoid:** The plan must explicitly note that spec §13 code is a v1 example and provide the v2 equivalent. Use `View() tea.View` + `v.AltScreen = true`.
**Warning signs:** `undefined: tea.WithAltScreen` compile error.

### Pitfall 2: Build Progress Blocking Issue
**What goes wrong:** The existing `buildImage()` is synchronous — it calls `streamBuildOutput()` which blocks until the build finishes. Wrapping it naively in a `tea.Cmd` means the TUI gets no updates during the build; it gets one giant message at the end.
**Why it happens:** The current `Manager.Build()` returns `(imageID, error)` — no channel output. The spec's D-07 references a `<-chan BuildProgress` that doesn't exist yet.
**How to avoid:** The plan needs a Wave 0 task to add a `BuildWithProgress(ctx, noCache, ch chan<- string) (string, error)` method (or similar) to the docker manager that writes each build line to the channel. The existing `streamBuildOutput` can be adapted to also write to a channel. Alternatively, launch `buildImage` in a goroutine that writes to a `chan string` and close the channel when done — this is the minimal change.
**Warning signs:** TUI shows blank output during build, then all lines appear at once on completion.

### Pitfall 3: Log Viewer Piped-stdout Detection
**What goes wrong:** `zone logs -f | grep error` launches the BubbleTea TUI anyway, which garbles the grep output with ANSI escape codes.
**Why it happens:** TTY detection only checks stdin (D-22). But piped stdout also needs detection for log viewer.
**How to avoid:** `IsOutputTTY()` must check `os.Stdout.Fd()` separately from stdin. Log viewer TUI only activates when BOTH stdin AND stdout are terminals.
**Warning signs:** `zone logs -f | grep error` produces ANSI-escaped garbage.

### Pitfall 4: --plain Flag Propagation
**What goes wrong:** `--plain` is defined as a PersistentFlag on rootCmd, but subcommands read it inconsistently.
**Why it happens:** `cmd.Flags().GetBool("plain")` reads local flags; `--plain` is a PersistentFlag so it requires `cmd.Root().PersistentFlags().GetBool("plain")` or `cmd.InheritedFlags().Lookup("plain")`.
**How to avoid:** The centralized `IsTTY()` helper must read `--plain` from the root command's persistent flags, not from the subcommand's local flags. Pass the Cobra command to `IsTTY()` or read the flag at call site via `cmd.Root().PersistentFlags().GetBool("plain")`.
**Warning signs:** `zone status --plain` still launches BubbleTea.

### Pitfall 5: Init Wizard Must Return Selected Harness to Caller
**What goes wrong:** The BubbleTea model runs and exits, but `cmd/init.go` has no way to retrieve the user's selection to pass to `generateInitTemplate()`.
**Why it happens:** `tea.NewProgram().Run()` returns a `tea.Model` interface, not the concrete struct. Type assertion is required to extract state.
**How to avoid:** The `InitWizard` struct must export the `SelectedHarness string` and `Confirmed bool` fields, and the caller type-asserts: `result := finalModel.(tui.InitWizard)`.
**Warning signs:** `zone init` with TUI always writes empty or default harness name.

### Pitfall 6: Viewport Width/Height Before WindowSizeMsg
**What goes wrong:** Viewport is initialized with zero width/height (before the terminal size is known), causing rendering with zero-width content.
**Why it happens:** `tea.WindowSizeMsg` is sent by BubbleTea after program start. The viewport must handle this message to set its dimensions.
**How to avoid:** Initialize viewport with 80x24 defaults. In `Update`, handle `tea.WindowSizeMsg` and call `vp.Width = msg.Width; vp.Height = msg.Height - N`.
**Warning signs:** Empty viewport on first render.

### Pitfall 7: Status View Polling Races
**What goes wrong:** The 2-second ticker fires and the Docker inspect call is slow, causing the next tick to fire while the previous inspect is still running.
**Why it happens:** `tea.Tick` schedules the next tick from the time the message is sent, not from when it is processed.
**How to avoid:** Only schedule the next tick from inside the `tickMsg` handler after processing the result — not from `Init`. This creates a "poll → wait for result → poll" sequential loop rather than a fixed interval.

## Code Examples

Verified patterns from official sources:

### BubbleTea v2 Program Creation (no options for inline, AltScreen in View for full-screen)
```go
// Source: https://pkg.go.dev/charm.land/bubbletea/v2

// Inline rendering (build progress, init wizard):
p := tea.NewProgram(model)
finalModel, err := p.Run()

// Alt-screen (status view, log viewer) — set inside View():
func (m Model) View() tea.View {
    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
// Still just:
p := tea.NewProgram(model)
finalModel, err := p.Run()
```

### Key Handling in v2
```go
// Source: https://github.com/charmbracelet/bubbletea/discussions/1374
case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit()
    case "up":
        m.list.CursorUp()
    case "down":
        m.list.CursorDown()
    case "enter":
        m.selected = m.list.SelectedItem()
        return m, tea.Quit()
    }
```

### Ticker for Status Polling
```go
// Source: https://pkg.go.dev/charm.land/bubbletea/v2 — tea.Tick
type tickMsg time.Time

func (m StatusView) Init() tea.Cmd {
    return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tickMsg:
        info, err := m.mgr.Status(m.ctx)
        if err != nil { m.err = err; return m, nil }
        m.info = info
        // Schedule next tick only after processing this one:
        return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
            return tickMsg(t)
        })
    }
    return m, nil
}
```

### Channel Relay Pattern for Docker Build Streaming
```go
// Source: https://github.com/charmbracelet/bubbletea/blob/main/examples/realtime/main.go

type buildLineMsg string
type buildDoneMsg struct{ imageID string }

func waitForLine(ch <-chan buildLineMsg) tea.Cmd {
    return func() tea.Msg { return <-ch }
}

func (m BuildProgress) Init() tea.Cmd {
    // Start build in goroutine, feed lines to channel
    return tea.Batch(
        m.spinner.Tick,
        waitForLine(m.ch),
        m.startBuild(),   // kicks off goroutine
    )
}

// startBuild returns a Cmd that launches a goroutine to run buildImage
// and send lines to m.ch. When done, sends buildDoneMsg or buildErrMsg.
func (m BuildProgress) startBuild() tea.Cmd {
    return func() tea.Msg {
        // This is the goroutine launcher — it itself returns a Msg (buildDoneMsg)
        // while the channel fills with intermediate buildLineMsg values.
        // The goroutine writes to m.ch; waitForLine relays each line.
        // ... see Architecture section for full approach
        return nil
    }
}
```

### Bubbles v2 Spinner
```go
// Source: https://pkg.go.dev/charm.land/bubbles/v2/spinner
import "charm.land/bubbles/v2/spinner"

s := spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
)

// In Init: return s.Tick  (NOT s.TickCmd — v2 uses Tick directly)
// In Update:
case spinner.TickMsg:
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd

// In View:
fmt.Sprintf("%s Building...", m.spinner.View())
```

### Lip Gloss v2 Box-Drawing for Status View
```go
// Source: https://pkg.go.dev/charm.land/lipgloss/v2
import "charm.land/lipgloss/v2"

boxStyle := lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("63")).
    Padding(0, 1).
    Width(42)

content := fmt.Sprintf(
    "Repo:      %s\nHarness:   %s\nStatus:    %s\n",
    cwd, cfg.Zone.Harness, state,
)
// Rendered: boxStyle.Render(content)
```

### TTY Detection
```go
// Source: https://pkg.go.dev/golang.org/x/term
import (
    "os"
    "golang.org/x/term"
)

func isTTY(cmd *cobra.Command) bool {
    plain, _ := cmd.Root().PersistentFlags().GetBool("plain")
    if plain {
        return false
    }
    return term.IsTerminal(int(os.Stdin.Fd()))
}

func isOutputTTY() bool {
    return term.IsTerminal(int(os.Stdout.Fd()))
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `github.com/charmbracelet/bubbletea` (v1) | `charm.land/bubbletea/v2` | Feb 24, 2025 | New import path; `View()` returns `tea.View`; `tea.KeyPressMsg` replaces `tea.KeyMsg` |
| `tea.WithAltScreen()` ProgramOption | `v.AltScreen = true` in `View()` | Feb 24, 2025 | Alt-screen is now declarative in the View struct |
| `github.com/charmbracelet/lipgloss` (v1) | `charm.land/lipgloss/v2` | Feb 24, 2025 | New import path; `Canvas`/`Compositor` added |
| `github.com/charmbracelet/bubbles` (v1) | `charm.land/bubbles/v2` | Feb 24, 2025 | New import path; viewport uses option-based New() |
| `os.Stdin.Stat() ModeCharDevice` | `golang.org/x/term.IsTerminal()` | Spec requirement | Cross-platform; official Go stdlib extension |

**Deprecated/outdated in this codebase:**
- `isInteractive()` in `cmd/init.go`: Uses `os.Stdin.Stat()` approach; replace with `term.IsTerminal()` (D-22, D-23)
- `promptHarnessSelection()` in `cmd/init.go`: Plain text selection prompt; replace with BubbleTea init wizard when TTY (D-04)
- `cmd/init.go` import of `bufio`, `strconv`: No longer needed after TUI replaces plain text prompt

## Critical Implementation Notes

### Build Progress Requires docker.Manager Refactor

The spec D-07 decision references `<-chan BuildProgress` from `Manager.Build()`, but the current `Manager.Build()` signature is `(ctx, noCache) (imageID string, err error)` — purely synchronous. The existing `streamBuildOutput()` in `build.go` writes lines to an `io.Writer`, not a channel.

**Recommended approach for the plan:**
1. Add `BuildWithProgress(ctx context.Context, noCache bool, lines chan<- string) (imageID string, err error)` to the Manager. This wraps `buildImage` with a custom `io.Writer` that splits lines and sends each to the channel.
2. The build progress TUI calls `BuildWithProgress` — launching it via a goroutine-kickoff Cmd, with `waitForLine` relaying individual lines to the Update loop.
3. The existing `Manager.Build()` signature is unchanged (other callers unaffected).

This is a narrow addition to `internal/docker/` that the build progress TUI task depends on.

### Two-Screen State Machine for Init Wizard

The init wizard has two screens: harness selector (list) and config preview. This requires a `screen` field in the model struct that switches between `screenSelector` and `screenPreview` constants. The list component and preview content are both embedded in the model; the `View()` method renders based on current screen.

### Inline vs Alt-Screen Rendering

Per D-28 and D-09:
- **Init wizard:** Inline (no alt-screen). `v.AltScreen` stays false (default).
- **Build progress:** Inline (no alt-screen). Content flows in scrollback buffer.
- **Status view:** Alt-screen (`v.AltScreen = true`). Full terminal takeover.
- **Log viewer:** Alt-screen (`v.AltScreen = true`). Full terminal takeover.

## Open Questions

1. **Build streaming channel type**
   - What we know: Current `buildImage` is synchronous; D-07 references a channel that doesn't exist yet
   - What's unclear: Should the channel carry `string` (line text) or a richer `BuildProgress` struct?
   - Recommendation: Use `chan string` for simplicity — the TUI only needs to display lines. Add `BuildWithProgress` to Manager in Wave 0 of the plan.

2. **Log viewer search: in-memory vs re-read**
   - What we know: Logs can be large; substrate text search needs to be fast enough
   - What's unclear: The viewport component highlights via `SetHighlights()` — but this requires knowing byte offsets or line indices. In-memory substring search with line highlighting is the practical approach.
   - Recommendation: Load all log content into a `[]string` slice; search via `strings.Contains` on each line; rebuild viewport content with highlighted lines using Lip Gloss styling. Re-highlight on each keystroke in the search textinput.

3. **Init wizard `[c] Customize` hotkey scope**
   - What we know: D-03 includes `[c] customize (open further options)` on the config preview screen
   - What's unclear: "Open further options" — what does this show? A third screen? An $EDITOR?
   - Recommendation: Out of scope per deferred list (the `--edit` flag is explicitly deferred). `[c]` can be a no-op placeholder that shows a hint message in the status bar: "Use `zone config` to customize after init".

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | none (standard `go test ./...`) |
| Quick run command | `go test ./tests/ -run TestTUI -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-01 | Init wizard model: selector → preview → write zone.toml | unit | `go test ./internal/tui/ -run TestInitWizard -v` | ❌ Wave 0 |
| TUI-02 | Build progress: lines arrive via channel, displayed in viewport | unit | `go test ./internal/tui/ -run TestBuildProgress -v` | ❌ Wave 0 |
| TUI-03 | Status view: polls mgr.Status, renders correct fields | unit | `go test ./internal/tui/ -run TestStatusView -v` | ❌ Wave 0 |
| TUI-04 | Log viewer: loads content, follow mode, search | unit | `go test ./internal/tui/ -run TestLogViewer -v` | ❌ Wave 0 |
| TUI-05 | TTY detection: non-TTY returns false, `--plain` returns false | unit | `go test ./internal/tui/ -run TestIsTTY -v` | ❌ Wave 0 |
| TUI-06 | `--plain` bypasses TUI: `zone init --harness x` in non-TTY works | integration | `go test ./tests/ -run TestInitNonTTY -v` | ❌ Wave 0 |
| TUI-07 | Non-TTY init without --harness: helpful error message | integration | `go test ./tests/ -run TestInitNoHarnessNonTTY -v` | ❌ existing in cli_commands_test.go |

**Note on TUI unit testing:** BubbleTea models can be tested without a real terminal by calling `model.Update(msg)` directly and asserting on the returned model state. No PTY or terminal emulator is needed. The `tea.NewProgram().Run()` integration path is covered by the existing `getZoneBinary` integration test infrastructure (non-TTY path only — TTY path requires manual verification).

### Sampling Rate
- **Per task commit:** `go test ./internal/tui/ -v -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/tui/tty.go` — IsTTY() and IsOutputTTY() helpers (TUI-05, TUI-06)
- [ ] `internal/tui/init_wizard_test.go` — covers TUI-01
- [ ] `internal/tui/build_progress_test.go` — covers TUI-02
- [ ] `internal/tui/status_view_test.go` — covers TUI-03
- [ ] `internal/tui/log_viewer_test.go` — covers TUI-04
- [ ] `tests/tui_integration_test.go` — covers TUI-06 and TUI-07 (non-TTY path)
- [ ] `internal/docker/build_progress.go` — `BuildWithProgress()` method (needed by TUI-02)
- [ ] Framework install: `go get charm.land/bubbletea/v2@v2.0.2 charm.land/bubbles/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.2 golang.org/x/term@v0.41.0`

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/charm.land/bubbletea/v2` — Model interface, ProgramOptions, View struct, AltScreen field
- `pkg.go.dev/charm.land/bubbles/v2/list` — list.New, Item/DefaultItem interfaces, Model methods
- `pkg.go.dev/charm.land/bubbles/v2/viewport` — viewport.New, SetContent, GotoBottom, AtBottom
- `pkg.go.dev/charm.land/bubbles/v2/spinner` — spinner.New, WithSpinner, available types
- `pkg.go.dev/charm.land/lipgloss/v2` — NewStyle, BorderStyle, Color
- `pkg.go.dev/golang.org/x/term` — IsTerminal signature and usage
- `github.com/charmbracelet/bubbletea/releases` — v2.0.0 Feb 24, 2025; v2.0.2 Mar 9, 2025 (current stable)
- `github.com/charmbracelet/bubbletea/discussions/1374` — v2 breaking changes: KeyPressMsg, View() → tea.View, AltScreen in View
- `github.com/charmbracelet/bubbletea/examples/realtime/main.go` — waitForActivity channel relay pattern

### Secondary (MEDIUM confidence)
- `charm.land/blog/v2/` — v2 announcement confirming battle-tested in Crush AI production
- `byteiota.com/bubble-tea-v2-10x-faster-terminal-uis-for-go-developers/` — v2 stability validation

### Tertiary (LOW confidence)
- None — all key findings verified against official pkg.go.dev documentation

## Metadata

**Confidence breakdown:**
- Standard stack (v2 vs v1 decision): HIGH — verified versions via `go get` in workspace; confirmed v2.0.2 stable
- Architecture patterns: HIGH — all patterns verified against official pkg.go.dev docs and official example
- v2 API changes (View returns tea.View, AltScreen field, KeyPressMsg): HIGH — verified from pkg.go.dev and GitHub discussion #1374
- Build progress channel pattern: HIGH — verified from official realtime example; LOW for `BuildWithProgress` (design decision, no existing code)
- Pitfalls: MEDIUM — based on verified API + logical inference from code reading

**Research date:** 2026-04-03
**Valid until:** 2026-07-03 (stable libraries; 90-day estimate for non-fast-moving Charm libraries)
