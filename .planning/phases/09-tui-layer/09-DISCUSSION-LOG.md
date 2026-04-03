# Phase 9: TUI Layer - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-03
**Phase:** 09-tui-layer
**Areas discussed:** Init wizard scope, BubbleTea version, Alt-screen policy, Log viewer search
**Mode:** --auto (all selections auto-resolved)

---

## Init Wizard Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full spec-compliant wizard | Multi-screen: harness selection + config preview + hotkeys per spec §13 | ✓ |
| Minimal harness selector | Simple list selection + confirm, no config preview | |

**User's choice:** [auto] Full spec-compliant wizard (recommended default)
**Notes:** Spec §13 is explicit about the wizard flow including harness list with detection hints, config preview, and hotkeys ([s], [n], [c], [Enter], [q]). No reason to deviate.

---

## BubbleTea Version

| Option | Description | Selected |
|--------|-------------|----------|
| Defer to researcher | Let researcher verify v2 Cobra integration before committing | ✓ |
| Use v1 (stable) | Well-documented, proven patterns | |
| Use v2 (latest) | Newer API, Feb 2026 release | |

**User's choice:** [auto] Defer to researcher (recommended default)
**Notes:** STATE.md explicitly flags this as a concern: "BubbleTea v2.0.0 is only one month old (Feb 2026) — Cobra integration patterns for v2 need verification before Phase 9." Research should resolve this before planning.

---

## Alt-Screen Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Status + log viewer alt-screen; init + build inline | Interactive views that need hotkeys get full screen; transient views stay inline | ✓ |
| All TUI views alt-screen | Consistent approach but overkill for transient build progress | |
| All TUI views inline | Simpler but poor UX for status/logs with interactive hotkeys | |

**User's choice:** [auto] Status + log viewer alt-screen; init wizard + build progress inline (recommended default)
**Notes:** Status view has interactive hotkeys (q/r/s) and log viewer has search (/) — both benefit from alt-screen. Build progress is transient (ends when build completes). Init wizard is a one-shot flow that benefits from seeing terminal context above it.

---

## Log Viewer Search

| Option | Description | Selected |
|--------|-------------|----------|
| Include search | / keybinding for text search per spec §13 | ✓ |
| Defer search | Implement basic viewport first, add search later | |

**User's choice:** [auto] Include (recommended default)
**Notes:** Spec §13 explicitly shows `/ search` in the log viewer keybindings. Implementing without it would deviate from spec.

---

## Claude's Discretion

- Styling details (colors, borders, spacing)
- Spinner type for build progress
- Config preview layout details beyond spec mockups
- Search implementation approach (substring vs regex)
- Polling interval tuning for status view
- Mouse support options

## Deferred Ideas

- `--edit` flag on `zone config` — backlog
- `--schema` flag on `zone config` — backlog
- Mouse support in TUI views — backlog
- Theme/color customization — v2
