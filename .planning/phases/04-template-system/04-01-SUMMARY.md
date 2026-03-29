---
phase: 04-template-system
plan: 01
subsystem: templates
tags: [go-embed, dockerfile, docker, templates, naming, sha256]

# Dependency graph
requires:
  - phase: 03-cache-state
    provides: hash.go uses templates via embed; cache infrastructure used alongside naming
provides:
  - Three //go:embed string vars (DockerfileTmpl, EntrypointTmpl, ZoneBashrcTmpl) in pkg/templates
  - Full Dockerfile.tmpl with all spec section 11 conditional blocks
  - Full entrypoint.sh.tmpl with git-safe, git identity forwarding, exec PID-1 pattern
  - Full zone-bashrc.tmpl with PS1 prompt, aliases, shell RC, welcome message
  - hash.go migrated from fs.ReadFile to direct string var access
  - ContainerName/NetworkName/ContainerLabels functions in internal/docker/naming.go
  - ContainerSecurityFlags in internal/docker/errors.go
  - 9 naming/security tests in tests/naming_test.go
affects: [05-dockerfile-render, 06-lifecycle, 07-agent-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "//go:embed string var pattern (not embed.FS) for individual template files"
    - "Deterministic naming: zone-<sanitized-repo>-<16-hex-sha256> from absolute path"
    - "SecurityConfig struct for hardened container settings (no-new-privileges, CapDrop ALL)"

key-files:
  created:
    - pkg/templates/Dockerfile.tmpl
    - pkg/templates/entrypoint.sh.tmpl
    - pkg/templates/zone-bashrc.tmpl
    - internal/docker/naming.go
    - tests/naming_test.go
  modified:
    - pkg/templates/templates.go
    - internal/cache/hash.go
    - internal/docker/errors.go

key-decisions:
  - "embed.FS replaced with three individual string vars — allows direct string access without io/fs overhead"
  - "hash.go migrated simultaneously with templates to keep build passing (no intermediate broken state)"
  - "ContainerName uses filepath.Abs to ensure relative paths produce same result as absolute"
  - "SecurityConfig struct in errors.go (not naming.go) per plan structure; no sentinel errors needed yet"

patterns-established:
  - "Template string vars: import _ 'embed' (blank import) required for //go:embed with string vars"
  - "Naming: nameCleanRe=[^a-zA-Z0-9_.-] applied to repo base name, hash from full absolute path"

requirements-completed: [DOC-01, DOC-06, DOC-07]

# Metrics
duration: 10min
completed: 2026-03-29
---

# Phase 04 Plan 01: Template System Foundation Summary

**embed.FS replaced with three //go:embed string vars, all template files populated from spec section 11 verbatim, hash.go migrated to direct string access, and deterministic container naming with security flags implemented**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-29T08:00:00Z
- **Completed:** 2026-03-29T08:10:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Migrated pkg/templates from embed.FS to three individual string vars with blank embed import
- Populated all three template files with verbatim spec section 11 content (Dockerfile, entrypoint, zone-bashrc)
- Updated hash.go to use direct string var access, removing io/fs dependency
- All 4 existing hash tests continue to pass after migration
- Implemented ContainerName (zone-<sanitized>-<16-hex>), NetworkName (+"-net"), ContainerLabels (3 labels)
- Implemented ContainerSecurityFlags with no-new-privileges, CapDrop ALL, 5 CapAdd, PidsLimit 512
- 9 naming/security tests cover determinism, sanitization, format, abs path resolution, label count

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate embed pattern, populate template files, and fix hash.go** - `5339135` (feat)
2. **Task 2: Implement container naming, labels, and security flags** - `41fbb62` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `pkg/templates/templates.go` - Replaced embed.FS with three //go:embed string vars
- `pkg/templates/Dockerfile.tmpl` - Full spec section 11 content with all conditional blocks
- `pkg/templates/entrypoint.sh.tmpl` - Full spec content with git-safe, identity forwarding, exec pattern
- `pkg/templates/zone-bashrc.tmpl` - Full spec content with PS1, aliases, shell RC, welcome message
- `internal/cache/hash.go` - Migrated from fs.ReadFile to direct templates.DockerfileTmpl/EntrypointTmpl access
- `internal/docker/naming.go` - ContainerName, NetworkName, ContainerLabels per spec section 12
- `internal/docker/errors.go` - SecurityConfig struct and ContainerSecurityFlags function
- `tests/naming_test.go` - 9 tests covering all naming/security functions

## Decisions Made
- embed.FS replaced with individual string vars — the blank import (`import _ "embed"`) is required for Go's //go:embed directive with string vars; direct access avoids io/fs overhead in hash.go
- hash.go migrated in same task as templates to avoid an intermediate broken build state
- ContainerName uses `filepath.Abs` so relative and absolute paths always produce the same deterministic name
- SecurityConfig struct placed in errors.go per plan structure — no actual sentinel errors needed yet in this package

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Template string vars ready for Plan 02 render functions to call via `template.Must(template.New(...).Funcs(...).Parse(templates.DockerfileTmpl))`
- ContainerName/NetworkName/ContainerLabels ready for Phase 5/6 lifecycle commands
- ContainerSecurityFlags ready for Phase 5/6 container creation
- All 62 tests pass (no regressions)

## Self-Check: PASSED

All files verified present. All commits verified in git log.

---
*Phase: 04-template-system*
*Completed: 2026-03-29*
