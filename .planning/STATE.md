---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: completed
stopped_at: Completed 05-harness-plugin-system-02-PLAN.md
last_updated: "2026-03-29T19:26:42.735Z"
last_activity: 2026-03-27 — Phase 02 Plan 03 complete; zone config (annotated TOML + JSON) + zone validate (grouped errors, exit 2) + CFG-07/CFG-08 tests
progress:
  total_phases: 10
  completed_phases: 4
  total_plans: 13
  completed_plans: 12
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-26)

**Core value:** Run `zone launch` in any repo and get a sandboxed Docker workspace for your LLM coding agent, with zero manual Docker configuration.
**Current focus:** Phase 2 - Config Foundation (Plan 03 complete — phase done)

## Current Position

Phase: 2 of 10 (Config Foundation) — complete
Plan: 3 of 3 in current phase (complete)
Status: Phase 02 Plan 03 complete — zone config + zone validate CLI commands wired, integration tests passing
Last activity: 2026-03-27 — Phase 02 Plan 03 complete; zone config (annotated TOML + JSON) + zone validate (grouped errors, exit 2) + CFG-07/CFG-08 tests

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: none yet
- Trend: -

*Updated after each plan completion*
| Phase 01 P01 | 7 min | 2 tasks | 60 files |
| Phase 01 P02 | 2 min | 2 tasks | 7 files |
| Phase 02 P02 | 7min | 2 tasks | 5 files |
| Phase 02 P03 | 4min | 3 tasks | 6 files |
| Phase 03-cache-state P01 | 8min | 3 tasks | 4 files |
| Phase 03-cache-state P02 | 2min | 3 tasks | 4 files |
| Phase 03-cache-state P03 | 3 | 1 tasks | 2 files |
| Phase 04-template-system P01 | 10min | 2 tasks | 8 files |
| Phase 04-template-system P02 | 3min | 3 tasks | 6 files |
| Phase 05-harness-plugin-system P01 | 2min | 1 tasks | 4 files |
| Phase 05-harness-plugin-system P02 | 3 | 1 tasks | 7 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: 10 fine-granularity phases derived; dependency order enforced by compiler import graph
- [Roadmap]: TUI deferred to Phase 9 (after lifecycle validated); network sandboxing deferred to Phase 10 (Linux-only, high complexity)
- [Roadmap]: Phase 4 (Template System) and Phase 3 (Cache) are parallel to each other — both depend only on Phase 2
- [Phase 01]: cobra v1.10.2 pinned exactly as specified; all 15 RunE stubs use exact string "not implemented" for Phase 8 integration test detection
- [Phase 01-02]: homebrew_casks (not brews) per GoReleaser v2.10 deprecation; goreleaser snapshot in CI (not check) to actually cross-compile; cmd.SetVersion() pattern for ldflags vars; GORELEASER_CURRENT_TAG=v0.0.0-dev for untagged CI runs
- [Phase 02-01]: Two-phase TOML decode for harness sugar (string vs table conflict); HarnessName toml:"-" pattern post-decode; explicit XDG path avoids macOS UserConfigDir() ~/Library pitfall; *bool fields for nullable booleans enable merge semantics
- [Phase 02]: Section-aware Levenshtein uses lenient threshold for same-section bare comparisons; resolveSymlinkTarget() added for symlink detection when target doesn't exist
- [Phase 02]: Bool pointer merging via block-scope temp variable: mergeBoolPtr returns bool, assigned as &v to *bool field in MergedConfig
- [Phase 02]: renderAnnotatedTOML emits comment block above lists — inline TOML comments on array elements are invalid per spec pitfall 4
- [Phase 02]: zone validate loads global + repo separately to accumulate UnknownKeysError alongside valid partial config
- [Phase 02]: Integration tests use pre-built binary via sync.Once — avoids go run recompile per test
- [Phase 03-01]: ComputeHash takes version as string param to avoid main.go import graph violation
- [Phase 03-01]: readTrimmed returns ("", nil) for missing files — not-found is not an error
- [Phase 03-01]: Hash includes only Dockerfile.tmpl + entrypoint.sh.tmpl per spec; zone-bashrc.tmpl excluded
- [Phase 03-02]: zone clean warns but proceeds if lock held — avoids deadlock on crashed prior process
- [Phase 03-02]: EnsureGitignore as standalone function — operates on cwd, independent of .zone/ existence
- [Phase 03-02]: Stale lock: pid<=0 treated as stale — prevents permanent block from corrupted PID file
- [Phase 03-cache-state]: errors.Is in main.go traverses wrapped error chain — ErrLockContention wrapped via %w in Acquire() is correctly detected without custom Unwrap
- [Phase 03-cache-state]: Exit code 5 check placed before generic os.Exit(1) in main.go — ordering is critical for correct mapping; full binary e2e deferred to Phase 6 when zone launch calls Lock.Acquire()
- [Phase 04-template-system]: embed.FS replaced with three individual string vars — allows direct string access without io/fs overhead
- [Phase 04-template-system]: ContainerName uses filepath.Abs so relative and absolute paths always produce same deterministic name
- [Phase 04-template-system]: hash.go migrated simultaneously with templates to keep build passing with no intermediate broken state
- [Phase 04-template-system]: templateFuncs() and injectGenerationComment() defined once in dockerfile.go, shared by entrypoint.go and shellrc.go
- [Phase 04-template-system]: DetectGitIdentity() both-or-nothing: partial git config returns forward=false, empty strings for name/email
- [Phase 05-harness-plugin-system]: NodeVersion/PythonVersion are NOT Harness interface methods — they come from MergedConfig.Harness per RESEARCH.md anti-patterns
- [Phase 05-harness-plugin-system]: Get() wraps Validate() error with harness name prefix; placeholder stubs in harness.go for Plan 02 types keep plan compilable independently
- [Phase 05-harness-plugin-system]: Cross-harness validation order: foreign-key errors before stub 'not implemented' error; aider owns python_version; custom checks skip_permissions before entrypoint_command

### Pending Todos

None yet.

### Blockers/Concerns

- [Research]: BubbleTea v2.0.0 is only one month old (Feb 2026) — Cobra integration patterns for v2 need verification before Phase 9
- [Research]: Docker + nftables interaction on Linux distros where iptables is nftables-backed needs integration testing before Phase 10
- [Research]: macOS SSH_AUTH_SOCK domain sockets cannot be bind-mounted — needs explicit error surfacing (Phase 7)
- [Research]: Rootless Docker is incompatible with host-side iptables — needs clear error when detected (Phase 10)

## Session Continuity

Last session: 2026-03-29T19:26:42.732Z
Stopped at: Completed 05-harness-plugin-system-02-PLAN.md
Resume file: None
