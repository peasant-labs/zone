# Phase 9: TUI Layer - Context

**Gathered:** 2026-04-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the BubbleTea interactive TUI layer for zone. Replace plain text output paths with BubbleTea views when running in a TTY. Four TUI components: init wizard (harness selection + config preview), build progress (Docker build streaming), status view (live container status), log viewer (with follow and search). Wire TTY auto-detection so BubbleTea activates only when appropriate, and respect `--plain` to force-disable TUI. Non-TTY `zone init` without `--harness` errors with helpful message. Terminal state must be fully restored after any TUI session, including forced exits and panics.

</domain>

<decisions>
## Implementation Decisions

### Init Wizard (TUI-01)
- **D-01:** Full spec-compliant wizard per spec §13: BubbleTea list selector with harness names, descriptions, and auto-detection hints (e.g., "* detected" for claude-code when `.claude/` exists)
- **D-02:** After harness selection, show config preview screen with harness-specific details (base image, packages, network mode, auth, skip_permissions status)
- **D-03:** Config preview hotkeys: [s] toggle skip_permissions, [n] network sandboxing, [c] customize (open further options), [Enter] confirm and write, [q] cancel
- **D-04:** Replace current `promptHarnessSelection()` in `cmd/init.go` with BubbleTea model when TTY detected; keep text fallback for non-TTY
- **D-05:** `zone launch` without `zone.toml` and without `--harness` in TTY: launch init wizard inline (same BubbleTea flow), write config, then proceed to build/launch per spec §3.6

### Build Progress (TUI-02)
- **D-06:** BubbleTea viewport with spinner, streaming Docker build output line-by-line
- **D-07:** Subscribe to `<-chan BuildProgress` from `Manager.Build()` via `tea.Cmd` that reads from the channel
- **D-08:** On build complete, transition to TTY attach (or print container ID if `--headless`)
- **D-09:** Inline rendering (not alt-screen) — build progress is transient and should flow in the terminal output
- **D-10:** Keybindings: down to scroll, Ctrl+C to cancel build

### Status View (TUI-03)
- **D-11:** Alt-screen BubbleTea model with box-drawn status display per spec §13 layout
- **D-12:** Polls container status every 2 seconds via Docker SDK `ContainerInspect`
- **D-13:** Display fields: Repo, Harness, Container name, Status (with uptime), Image ID, Network mode, Port mappings, Resource limits, Mounts
- **D-14:** Interactive hotkeys: q quit, r restart, s stop
- **D-15:** `--json` flag bypasses BubbleTea entirely — prints raw JSON to stdout (existing Phase 8 behavior preserved)

### Log Viewer (TUI-04)
- **D-16:** Alt-screen BubbleTea viewport with auto-scroll and search
- **D-17:** `--follow` starts in follow mode (auto-scroll to bottom); without `--follow`, starts paused at the end
- **D-18:** `/ search` keybinding for text search within logs per spec §13
- **D-19:** `--build` loads from `.zone/logs/last_build.log` instead of live container logs
- **D-20:** When stdout is piped, output plain text (no TUI chrome) — `zone logs -f | grep error` works correctly
- **D-21:** Keybindings: up/down scroll, / search, f toggle follow, q quit

### TTY Detection (TUI-05, TUI-06)
- **D-22:** Use `golang.org/x/term` package's `term.IsTerminal()` on stdin fd per spec §3.5, replacing current `os.Stdin.Stat()` approach in `cmd/init.go`
- **D-23:** Centralize TTY check as a helper in `cmd/` or `internal/tui/` — all commands that conditionally use TUI call this single function
- **D-24:** `--plain` flag (already defined in root.go) force-disables TUI even when TTY detected
- **D-25:** When TUI disabled (non-TTY or `--plain`), commands fall back to existing plain text output from Phase 8

### Non-TTY Init Error (TUI-07)
- **D-26:** `zone init` without `--harness` in non-TTY: error with "Interactive mode requires a terminal. Use `--harness <name>` for non-interactive init." — already implemented in Phase 8, just needs to use the centralized TTY check

### Terminal Restore / Panic Recovery
- **D-27:** All BubbleTea programs wrapped in a deferred panic recovery that restores terminal state before re-panicking
- **D-28:** Use `tea.WithAltScreen()` only for status view and log viewer; init wizard and build progress use inline rendering
- **D-29:** BubbleTea's built-in cleanup handles normal exits; panic handler covers abnormal exits

### BubbleTea Version
- **D-30:** Defer version choice (v1 vs v2) to researcher — STATE.md flags "BubbleTea v2.0.0 is only one month old (Feb 2026) — Cobra integration patterns for v2 need verification." Researcher must verify v2 stability and Cobra integration patterns before planning proceeds.

### Cobra Integration Pattern
- **D-31:** All BubbleTea models follow the spec §13 integration pattern: check `--json`/`--plain`/non-TTY first, fall back to plain text; otherwise create TUI model and run `tea.NewProgram(model).Run()`
- **D-32:** Each TUI model lives in `internal/tui/` and is imported by the corresponding `cmd/` file

### Claude's Discretion
- Exact styling choices (colors, borders, spacing) within Lip Gloss — as long as consistent
- Spinner type/style for build progress
- Config preview layout details beyond what spec §13 shows
- Search implementation approach in log viewer (substring vs regex)
- Polling interval for status view (spec says 2s, can adjust if needed)
- Whether to use `tea.WithAltScreen()` options like `tea.WithMouseCellMotion()`

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### TUI component specifications
- `zone-spec.md` §13 (lines 1306-1433) — Complete BubbleTea TUI component specs: init wizard layout, build progress display, status view layout, log viewer layout, ls table format, Cobra integration pattern with Go code example
- `zone-spec.md` §3.5 (lines 112-119) — TTY detection: `term.IsTerminal()`, BubbleTea when TTY, plain text when not, `--plain` override

### Init wizard context
- `zone-spec.md` §3.6 (lines 121-129) — `zone launch` without `zone.toml` behavior: TTY triggers inline init wizard
- `zone-spec.md` §3.1-3.2 (lines 52-85) — Command table showing which commands have BubbleTea views

### Build progress context
- `zone-spec.md` §9 (lines 778-818) — Docker SDK build streaming, `<-chan BuildProgress` pattern

### JSON bypass
- `zone-spec.md` §3.13 (lines 244-248) — `--json` bypasses BubbleTea entirely

### Project structure
- `zone-spec.md` §7 (lines 645-705) — File layout showing `internal/tui/` directory with all four model files

### Requirements
- `.planning/REQUIREMENTS.md` — TUI-01 through TUI-07

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/tui/init_wizard.go` — Stub file (package declaration only), ready for implementation
- `internal/tui/build_progress.go` — Stub file (package declaration only), ready for implementation
- `internal/tui/status_view.go` — Stub file (package declaration only), ready for implementation
- `internal/tui/log_viewer.go` — Stub file (package declaration only), ready for implementation
- `cmd/init.go` — Has `detectHarnessHints()`, `generateInitTemplate()`, `isInteractive()`, `promptHarnessSelection()` — detection logic reusable, selection to be replaced with BubbleTea
- `cmd/root.go` — `--plain` flag already defined at line 50, needs wiring to TUI gating logic
- `cmd/launch.go` — Signal context pattern (`signal.NotifyContext`) already established
- `cmd/status.go`, `cmd/logs.go` — Existing plain text output paths to be preserved as fallback

### Established Patterns
- Cobra command structure: `var xxxCmd` + `init()` for flags
- Manager construction: `config.LoadMerged` → `cache.New` → `docker.NewManager`
- JSON bypass: `json.MarshalIndent` + `fmt.Fprintln(cmd.OutOrStdout())` pattern from `cmd/config.go`
- Signal handling: `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` in all Docker-calling commands

### Integration Points
- `cmd/init.go` RunE → conditionally launch `tui.NewInitWizard()` when TTY
- `cmd/launch.go` RunE → conditionally launch `tui.NewBuildProgress()` during build when TTY
- `cmd/status.go` RunE → conditionally launch `tui.NewStatusView()` when TTY and no `--json`
- `cmd/logs.go` RunE → conditionally launch `tui.NewLogViewer()` when TTY and no `--json` and stdout not piped
- `cmd/launch.go` → inline init wizard when no zone.toml + no --harness + TTY

</code_context>

<specifics>
## Specific Ideas

No specific requirements — all decisions auto-selected from recommended defaults based on spec §13.

</specifics>

<deferred>
## Deferred Ideas

- `--edit` flag on `zone config` (opens $EDITOR) — backlog (from Phase 8 deferred)
- `--schema` flag on `zone config` — backlog (from Phase 8 deferred)
- `--from-devcontainer` migration on `zone init` — v2 backlog (from Phase 8 deferred)
- Mouse support in TUI views — backlog if requested
- Theme/color customization in config — v2 feature

</deferred>

---

*Phase: 09-tui-layer*
*Context gathered: 2026-04-03*
