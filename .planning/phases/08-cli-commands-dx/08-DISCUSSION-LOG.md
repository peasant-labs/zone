# Phase 8: CLI Commands & DX - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-30
**Phase:** 08-cli-commands-dx
**Areas discussed:** init scope, ls location, logs scope, status scope, exit codes, signal handling, remediation hints, help text, --port flag
**Mode:** Auto (all recommended defaults selected)

---

## init Command Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Non-interactive only | `--harness` + `--set` flags, error without --harness | ✓ |
| Basic text-mode wizard | Simple stdin prompts for harness selection | |
| Full TUI wizard | BubbleTea interactive (Phase 9) | |

**User's choice:** [auto] Non-interactive only (recommended default — TUI is Phase 9)

---

## ls Implementation Location

| Option | Description | Selected |
|--------|-------------|----------|
| Manager.List() method | Docker query in Manager, cmd formats | ✓ |
| Standalone in cmd/ls.go | Direct Docker client query in command | |

**User's choice:** [auto] Manager.List() method (recommended default — consistent pattern)

---

## logs Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Plain text + --follow + --build | SDK streaming, cache file read | ✓ |
| Plain text only (no --follow) | One-shot log dump | |

**User's choice:** [auto] Full plain text with --follow streaming

---

## status Scope

| Option | Description | Selected |
|--------|-------------|----------|
| One-shot plain text + --json | ContainerInspect, format, print | ✓ |
| Live polling plain text | Refresh every 2s (Phase 9 territory) | |

**User's choice:** [auto] One-shot plain text (recommended default — live view is Phase 9)

---

## Exit Code Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| errors.Is chain in main.go | Extend existing pattern | ✓ |
| Custom error type with code | Each error carries its exit code | |

**User's choice:** [auto] errors.Is chain (recommended default — existing pattern)

---

## Signal Handling

| Option | Description | Selected |
|--------|-------------|----------|
| signal.NotifyContext in all Docker commands | Per spec §9 | ✓ |
| Only in launch command | Minimal change | |

**User's choice:** [auto] All Docker commands (recommended default — spec says all)

---

## Remediation Hints

| Option | Description | Selected |
|--------|-------------|----------|
| cmd/errors.go mapper function | mapError returns message + code | ✓ |
| Custom error type with hint field | Embed hint in error | |

**User's choice:** [auto] Mapper function (recommended default — internal packages stay clean)

---

## Help Text

| Option | Description | Selected |
|--------|-------------|----------|
| All 15 commands | Long field with 2-4 examples each | ✓ |
| Only new commands | Just init, ls, logs, status | |

**User's choice:** [auto] All 15 commands (recommended default — DX-09 says all)

---

## --port/-P Flag

| Option | Description | Selected |
|--------|-------------|----------|
| Add to launch command | Repeatable string flag, merge with config ports | ✓ |
| Defer further | Wait for Phase 9 | |

**User's choice:** [auto] Add now (recommended default — was deferred from Phase 7 specifically to Phase 8)

---

## Deferred Ideas

- TUI BubbleTea views → Phase 9
- --edit, --schema flags on config → backlog
- --from-devcontainer → v2
