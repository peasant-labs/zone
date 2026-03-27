---
phase: 01-project-scaffold
plan: 01
subsystem: infra
tags: [go, cobra, cli, module-scaffold]

# Dependency graph
requires: []
provides:
  - "Go module github.com/peasant-labs/zone with cobra v1.10.2"
  - "main.go entry point with ldflags version/commit/date vars"
  - "cmd/root.go with 15 wired Cobra subcommands and 4 global flags"
  - "All 15 cmd/*.go stubs returning fmt.Errorf(\"not implemented\")"
  - "31 internal package stubs across config, cache, docker, network, harness, tui"
  - "pkg/templates/templates.go with go:embed FS and 3 placeholder .tmpl files"
  - "6 test stubs in tests/ package"
affects:
  - "02-config-layer"
  - "03-cache-layer"
  - "04-template-system"
  - "all subsequent phases"

# Tech tracking
tech-stack:
  added:
    - "github.com/spf13/cobra v1.10.2"
    - "github.com/spf13/pflag v1.0.9 (transitive)"
    - "github.com/inconshreveable/mousetrap v1.1.0 (transitive, Windows)"
  patterns:
    - "Cobra command pattern: package-level var xyzCmd = &cobra.Command{...}"
    - "RunE stub returns fmt.Errorf(\"not implemented\") exactly"
    - "Internal packages: package declaration + doc comment only, no imports"
    - "go:embed *.tmpl for templates FS"
    - "tests/ package uses package tests, no internal imports"

key-files:
  created:
    - "go.mod - module definition with cobra v1.10.2 dependency"
    - "main.go - entry point with version/commit/date ldflags vars"
    - "cmd/root.go - Cobra root command with 15 subcommands and 4 global flags"
    - "cmd/launch.go - representative stub with up alias"
    - "pkg/templates/templates.go - go:embed FS for Dockerfile/entrypoint/shellrc templates"
    - "internal/docker/errors.go - docker package stub"
    - "internal/config/types.go - config package stub"
  modified: []

key-decisions:
  - "Used cobra v1.10.2 (not latest v1.9.x/v2) as specified in plan"
  - "Test stubs in tests/ use package tests with no internal imports (stubs have no real code yet)"
  - "Template files are non-empty comment placeholders so go:embed compiles without errors"
  - "15 cmd/*.go files each use fmt.Errorf(\"not implemented\") exactly — required by Phase 8 integration tests"

patterns-established:
  - "Cobra stub pattern: var xyzCmd = &cobra.Command{Use, Aliases, Short, RunE: fmt.Errorf(\"not implemented\")}"
  - "Internal stub pattern: doc comment + package declaration only"
  - "Template embedding: //go:embed *.tmpl on var FS embed.FS"

requirements-completed:
  - DX-10

# Metrics
duration: 7min
completed: 2026-03-27
---

# Phase 1 Plan 01: Go Module and CLI Skeleton Summary

**Compilable Go project with 15 wired Cobra subcommands, 8 internal package stubs, and embedded template FS — `go build ./...` and `go test ./...` both pass from clean checkout**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-03-27T01:12:40Z
- **Completed:** 2026-03-27T01:19:10Z
- **Tasks:** 2
- **Files modified:** 60 created, 0 modified

## Accomplishments

- Go module initialized at `github.com/peasant-labs/zone` with cobra v1.10.2; `go build ./...` and `go test ./...` pass
- All 15 Cobra subcommands wired with exact Short descriptions and aliases from UI-SPEC (up, down, list, log, st); all stubs return `fmt.Errorf("not implemented")`
- 31 internal package stubs and pkg/templates with go:embed FS establish full project tree from spec Section 7

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module and create Cobra CLI skeleton** - `0885e41` (feat)
2. **Task 2: Create all internal, pkg, and test stub files** - `390b2c1` (feat)

## Files Created/Modified

- `go.mod` - Module definition `github.com/peasant-labs/zone` with cobra v1.10.2
- `go.sum` - Dependency checksums
- `main.go` - Entry point with version/commit/date ldflags vars, calls cmd.Execute()
- `cmd/root.go` - Cobra root with Long description, 15 AddCommand calls, 4 PersistentFlags
- `cmd/{init,launch,join,exec,shell,build,stop,restart,ls,logs,clean,destroy,status,config,validate}.go` - 15 stub files
- `internal/config/{types,harness_config,config,global,merge,validate}.go` - config package stubs
- `internal/cache/{cache,hash,lock}.go` - cache package stubs
- `internal/docker/{manager,dockerfile,entrypoint,shellrc,naming,network,platform,errors}.go` - docker package stubs
- `internal/network/{firewall,rules,matcher}.go` - network package stubs
- `internal/harness/{harness,claude_code,opencode,gemini_cli,aider,codex_cli,custom}.go` - harness package stubs
- `internal/tui/{init_wizard,build_progress,status_view,log_viewer}.go` - tui package stubs
- `pkg/templates/templates.go` - go:embed FS pointing to 3 .tmpl files
- `pkg/templates/{Dockerfile,entrypoint.sh,zone-bashrc}.tmpl` - placeholder template files
- `tests/{config_merge,harness_validate,validate,naming,matcher,hash}_test.go` - 6 test stubs

## Decisions Made

- Used `cobra v1.10.2` exactly as specified (pinned version, not latest)
- Test stubs declare `package tests` with no imports — internal packages have no real code yet, adding imports would break compilation
- Template placeholder files use comment lines so they are non-empty and `go:embed *.tmpl` compiles cleanly
- Exact string `"not implemented"` used in all 15 RunE stubs as required by Phase 8 integration tests

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Full project skeleton is in place; `go build ./...` and `go test ./...` pass from clean checkout
- Phase 2 (Config Layer) can now import `internal/config` package and fill in the stubs
- Phase 3 (Cache) can import `internal/cache`; Phase 4 (Templates) can work with `pkg/templates`
- All 15 cmd/*.go stubs are ready to be filled in by Phase 5+ lifecycle implementation

---
*Phase: 01-project-scaffold*
*Completed: 2026-03-27*
