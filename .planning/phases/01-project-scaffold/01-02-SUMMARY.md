---
phase: 01-project-scaffold
plan: 02
subsystem: infra
tags: [goreleaser, golangci-lint, makefile, github-actions, ci, go, cross-compilation]

# Dependency graph
requires:
  - phase: 01-01
    provides: main.go with version/commit/date ldflags vars, go.mod module path, all stub packages compiled

provides:
  - GoReleaser v2 config cross-compiling 5 platform targets with Homebrew cask publishing
  - golangci-lint v2 config with standard + 6 additional linters
  - Makefile with 7 standard targets (build, test, lint, fmt, vet, clean, install)
  - GitHub Actions CI workflow with 4 parallel jobs (build-test, lint, govulncheck, goreleaser-check)
  - GitHub Actions release workflow triggered on v* tags

affects: [all future phases - CI validates every push; GoReleaser enables binary distribution]

# Tech tracking
tech-stack:
  added: [goreleaser v2.14.3, golangci-lint v2.11.4, github-actions]
  patterns:
    - GoReleaser v2 config format (version:2, homebrew_casks, CGO_ENABLED=0)
    - golangci-lint v2 config format (version:"2", linters.default:standard)
    - Cobra version injection via cmd.SetVersion() called from main()
    - GORELEASER_CURRENT_TAG=v0.0.0-dev env var for CI snapshot on untagged repos

key-files:
  created:
    - .goreleaser.yml
    - .golangci.yml
    - Makefile
    - .github/workflows/ci.yml
    - .github/workflows/release.yml
  modified:
    - cmd/root.go (added SetVersion function)
    - main.go (calls cmd.SetVersion before Execute)

key-decisions:
  - "homebrew_casks (not brews) - brews deprecated in GoReleaser v2.10"
  - "goreleaser release --snapshot --clean in CI (not goreleaser check alone) - snapshot actually cross-compiles, catching linker errors"
  - "GORELEASER_CURRENT_TAG=v0.0.0-dev env var on goreleaser CI step - prevents no-tags-found error on fresh repo"
  - "cmd.SetVersion() pattern - allows main.go ldflags vars to be consumed by cobra, satisfying unused linter"
  - "4 parallel independent CI jobs - build-test, lint, govulncheck, goreleaser-check all run concurrently"
  - "HOMEBREW_TAP_TOKEN secret required in GitHub repo settings for release workflow to publish Homebrew cask"

patterns-established:
  - "Pattern: Version injection via ldflags -> cmd.SetVersion() -> cobra rootCmd.Version"
  - "Pattern: GoReleaser snapshot in CI with GORELEASER_CURRENT_TAG for untagged branches"

requirements-completed: [DX-10]

# Metrics
duration: 2min
completed: 2026-03-27
---

# Phase 1 Plan 02: Toolchain Configuration Summary

**GoReleaser v2 cross-compiling 5 targets (linux/darwin/windows amd64+arm64 minus windows/arm64) with GitHub Actions 4-job CI pipeline, golangci-lint v2 standard+6 linters, and 7-target Makefile**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-27T01:23:08Z
- **Completed:** 2026-03-27T01:25:36Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- GoReleaser v2 config validated with `goreleaser check` and snapshot cross-compiles all 5 platform targets to `dist/`
- golangci-lint v2 passes with zero issues using default:standard baseline plus errorlint, errname, bodyclose, contextcheck, gocritic, misspell
- Makefile provides build/test/lint/fmt/vet/clean/install with hard tab indentation
- CI pipeline: 4 parallel jobs cover build+test, lint, vulnerability scan, and goreleaser snapshot
- Release pipeline: triggers on v* tags, runs goreleaser --clean with GITHUB_TOKEN and HOMEBREW_TAP_TOKEN

## Task Commits

Each task was committed atomically:

1. **Task 1: Create GoReleaser, golangci-lint, and Makefile configs** - `f1d1a98` (feat)
2. **Task 2: Create GitHub Actions CI and release workflows** - `34c55f7` (feat)

**Plan metadata:** (docs commit below)

## Files Created/Modified

- `.goreleaser.yml` - GoReleaser v2 config: 5 platform targets, CGO_ENABLED=0, homebrew_casks tap, ldflags for version/commit/date
- `.golangci.yml` - golangci-lint v2 config: default:standard + 6 linters
- `Makefile` - 7 targets with hard tab indentation; build outputs to bin/zone, clean removes bin/ and dist/
- `.github/workflows/ci.yml` - 4 parallel jobs on push/PR to any branch, ubuntu-latest, Go 1.24
- `.github/workflows/release.yml` - v* tag trigger, GoReleaser release, HOMEBREW_TAP_TOKEN for tap publishing
- `cmd/root.go` - Added SetVersion() function to accept ldflags-injected values
- `main.go` - Calls cmd.SetVersion(version, commit, date) before Execute()

## Decisions Made

- Used `homebrew_casks` (not `brews`) per GoReleaser v2.10 deprecation
- CI goreleaser job runs `release --snapshot --clean` (not just `goreleaser check`) to actually cross-compile and catch linker errors
- Set `GORELEASER_CURRENT_TAG=v0.0.0-dev` env var on goreleaser CI step to prevent "couldn't find any tags" failure on fresh repos
- Added `cmd.SetVersion()` helper and call it from main.go so the ldflags vars (`version`, `commit`, `date`) are consumed by the program, satisfying the `unused` linter

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed unused variable linter failure for version/commit/date ldflags vars**

- **Found during:** Task 1 (golangci-lint run verification)
- **Issue:** `golangci-lint run ./...` failed with 3 "var X is unused" errors for version, commit, date in main.go. These vars exist for GoReleaser ldflags injection but were never read by the program.
- **Fix:** Added `SetVersion(version, commit, date string)` function to `cmd/root.go` that sets `rootCmd.Version`. Updated `main.go` to call `cmd.SetVersion(version, commit, date)` before `cmd.Execute()`. This wires the ldflags-injected values into cobra's built-in `--version` flag.
- **Files modified:** `cmd/root.go`, `main.go`
- **Verification:** `golangci-lint run ./...` exits 0 with "0 issues."
- **Committed in:** `f1d1a98` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Auto-fix necessary for linter correctness. No scope creep — SetVersion is the idiomatic way to expose GoReleaser ldflags vars to cobra.

## Issues Encountered

- goreleaser binary was not pre-installed in the environment. Installed via `go install github.com/goreleaser/goreleaser/v2@latest` (Rule 3 - blocking issue auto-resolved). goreleaser v2.14.3 installed successfully to ~/go/bin.

## User Setup Required

**HOMEBREW_TAP_TOKEN** secret must be configured in GitHub repository settings before the release workflow can publish the Homebrew cask to `peasant-labs/homebrew-tap`. The CI snapshot job does NOT need this token. Steps:
1. Create a GitHub Personal Access Token with `repo` scope on the `peasant-labs/homebrew-tap` repository
2. Add it as a repository secret named `HOMEBREW_TAP_TOKEN` in the `peasant-labs/zone` repository settings

## Next Phase Readiness

- Complete toolchain ready: every push to any branch triggers 4-job CI validation
- GoReleaser snapshot verified to cross-compile all 5 targets from Linux runner
- Phase 1 Project Scaffold fully complete (plans 01 and 02 done)
- Phase 2 (Docker Integration) can begin

---
*Phase: 01-project-scaffold*
*Completed: 2026-03-27*
