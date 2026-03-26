# Phase 1: Project Scaffold - Research

**Researched:** 2026-03-26
**Domain:** Go module initialization, Cobra CLI scaffold, GitHub Actions CI, GoReleaser distribution
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- Go module path: `github.com/peasant-labs/zone`
- Minimum Go version: 1.24
- CI: GitHub Actions on push and PR, Linux-only runners (ubuntu-latest)
- CI checks: `go build ./...`, `go test ./...`, `golangci-lint`, `goreleaser check`, `govulncheck`
- GoReleaser snapshot build in CI to verify cross-compilation for all platforms
- Create ALL directories and packages from spec Section 7 with stub files
- Each stub file has package declaration and doc comment
- All `cmd/*` files register Cobra subcommands with placeholder `RunE` returning "not implemented"
- `zone --help` shows all commands from day one
- Makefile targets: `build`, `test`, `lint`, `fmt`, `vet`, `clean`, `install`
- Platforms: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- Homebrew tap: `peasant-labs/homebrew-tap`
- No Docker image of the CLI

### Claude's Discretion

- golangci-lint configuration (which linters to enable)
- Archive format details (tar.gz vs zip per platform)
- Exact Makefile target implementations
- Stub file doc comments and placeholder text
- CI workflow naming and job structure

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DX-10 | GoReleaser configuration for binary distribution | GoReleaser v2.14.3, `homebrew_casks` for tap, `goreleaser check` CI step |
</phase_requirements>

---

## Summary

Phase 1 establishes a complete Go project scaffold: `go.mod`, all packages from spec Section 7 with stub files, a working Cobra CLI that prints help with all subcommands, CI that validates every push, and GoReleaser config that verifies cross-compilation. No functional logic is written here — the goal is that `go build ./...`, `go test ./...`, and `goreleaser check` all pass from a clean checkout.

The primary complexity is breadth: the spec defines ~40 files across 8 packages plus 15 `cmd/` files. Each stub must have a valid package declaration and doc comment. The Cobra subcommand wiring in `cmd/root.go` must be complete so `zone --help` reflects the full command surface. GoReleaser must use `homebrew_casks` (not the deprecated `brews`) for the Homebrew tap.

GoReleaser v2 uses a `version: 2` header in `.goreleaser.yml`. The `homebrew_casks` key replaces the deprecated `brews` key as of v2.10. golangci-lint v2 uses a `version: "2"` header and `linters.default: standard` — the installed version on this machine is v2.11.4.

**Primary recommendation:** Wire all 15 Cobra subcommands in `cmd/root.go` on day one, use `goreleaser --snapshot --clean` in CI (not `goreleaser check` alone), and use `homebrew_casks` not `brews` in `.goreleaser.yml`.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go toolchain | 1.24 (min), 1.25.5 (installed) | Language runtime | Locked in CONTEXT.md |
| github.com/spf13/cobra | v1.10.2 | CLI subcommand framework | Spec-mandated; industry standard for Go CLIs |
| GoReleaser | v2.14.3 | Cross-platform binary release | Spec-mandated (DX-10) |
| golangci-lint | v2.11.4 (installed) | Static analysis runner | CI requirement from CONTEXT.md |
| golang.org/x/vuln (govulncheck) | v1.1.4 | Vulnerability scanning | CI requirement from CONTEXT.md |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/charmbracelet/bubbletea | v1.3.10 | TUI (stub imports only in Phase 1) | Needed for `internal/tui` stub package declaration |
| github.com/charmbracelet/lipgloss | v1.1.0 | TUI styling (stub imports only) | Paired with BubbleTea |
| github.com/BurntSushi/toml | v1.6.0 | TOML parsing (stub only) | Spec-mandated for config system |
| github.com/docker/docker | v28.5.2+incompatible | Docker SDK (stub only) | Spec-mandated for docker package |

**Version verification:** All versions confirmed via Go module proxy on 2026-03-26. Versions are current as of that date.

**Installation:**
```bash
go mod init github.com/peasant-labs/zone
go get github.com/spf13/cobra@v1.10.2
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/lipgloss@v1.1.0
go get github.com/BurntSushi/toml@v1.6.0
go get github.com/docker/docker@v28.5.2+incompatible
go mod tidy
```

> Note: Phase 1 stubs only need package declarations and doc comments. Stub files for packages like `internal/docker` and `internal/tui` should NOT import their heavy dependencies — those imports belong in the implementation phases. Only `cmd/` files need Cobra imported to register subcommands.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Cobra v1.10.2 | Cobra v2.x (if released) | v2.x series is unverified; proxy confirms v1.10.2 is latest stable |
| homebrew_casks | brews (deprecated) | brews deprecated in GoReleaser v2.10; casks is the current approach |
| golangci-lint default: standard | custom linter list | standard set is safe for a new project; add linters incrementally |

---

## Architecture Patterns

### Recommended Project Structure

Exactly as defined in spec Section 7. Reproduced here for planner reference:

```
zone/
├── go.mod
├── go.sum
├── main.go                       # entry point, version vars, Cobra root
├── Makefile
├── .goreleaser.yml
├── README.md
├── cmd/
│   ├── root.go                   # root command + global flags + AddCommand calls
│   ├── init.go
│   ├── launch.go
│   ├── join.go
│   ├── exec.go
│   ├── shell.go
│   ├── build.go
│   ├── stop.go
│   ├── restart.go
│   ├── ls.go
│   ├── logs.go
│   ├── clean.go
│   ├── destroy.go
│   ├── status.go
│   ├── config.go
│   └── validate.go
├── internal/
│   ├── config/
│   │   ├── types.go
│   │   ├── harness_config.go
│   │   ├── config.go
│   │   ├── global.go
│   │   ├── merge.go
│   │   └── validate.go
│   ├── cache/
│   │   ├── cache.go
│   │   ├── hash.go
│   │   └── lock.go
│   ├── docker/
│   │   ├── manager.go
│   │   ├── dockerfile.go
│   │   ├── entrypoint.go
│   │   ├── shellrc.go
│   │   ├── naming.go
│   │   ├── network.go
│   │   ├── platform.go
│   │   └── errors.go
│   ├── network/
│   │   ├── firewall.go
│   │   ├── rules.go
│   │   └── matcher.go
│   ├── harness/
│   │   ├── harness.go
│   │   ├── claude_code.go
│   │   ├── opencode.go
│   │   ├── gemini_cli.go
│   │   ├── aider.go
│   │   ├── codex_cli.go
│   │   └── custom.go
│   └── tui/
│       ├── init_wizard.go
│       ├── build_progress.go
│       ├── status_view.go
│       └── log_viewer.go
├── pkg/
│   └── templates/
│       ├── templates.go
│       ├── Dockerfile.tmpl
│       ├── entrypoint.sh.tmpl
│       └── zone-bashrc.tmpl
└── tests/
    ├── config_merge_test.go
    ├── harness_validate_test.go
    ├── validate_test.go
    ├── naming_test.go
    ├── matcher_test.go
    └── hash_test.go
```

### Pattern 1: Cobra Root Command with Stub Subcommands

All 15 subcommands must be registered in `cmd/root.go` via `rootCmd.AddCommand()`. Each `cmd/*.go` file declares a package-level `*cobra.Command` and registers it.

```go
// cmd/root.go
package cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "zone",
    Short: "Sandboxed Docker workspaces for LLM coding agents",
    Long:  `Zone generates and manages Docker workspaces for LLM coding agents.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.AddCommand(
        initCmd,
        launchCmd,
        joinCmd,
        execCmd,
        shellCmd,
        buildCmd,
        stopCmd,
        restartCmd,
        lsCmd,
        logsCmd,
        cleanCmd,
        destroyCmd,
        statusCmd,
        configCmd,
        validateCmd,
    )
    // Global flags
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Increase output verbosity")
    rootCmd.PersistentFlags().Bool("debug", false, "Maximum verbosity")
    rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output")
    rootCmd.PersistentFlags().Bool("plain", false, "Disable TUI, use plain text output")
}
```

```go
// cmd/launch.go — typical stub pattern
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
    Use:     "launch",
    Aliases: []string{"up"},
    Short:   "Build (if needed) and run the container",
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("not implemented")
    },
}
```

```go
// main.go
package main

import (
    "os"
    "github.com/peasant-labs/zone/cmd"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Pattern 2: Stub Internal Package Files

Each internal package stub file needs only a package declaration and a doc comment. No imports unless the package has zero-value types to declare (wait for implementation phases).

```go
// internal/config/types.go

// Package config provides TOML configuration parsing and merging for zone.
package config
```

### Pattern 3: GoReleaser v2 Configuration

```yaml
# .goreleaser.yml
version: 2

builds:
  - id: zone
    main: .
    binary: zone
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: zone
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: "checksums.txt"

homebrew_casks:
  - name: zone
    ids:
      - zone
    repository:
      owner: peasant-labs
      name: homebrew-tap
    homepage: "https://github.com/peasant-labs/zone"
    description: "Sandboxed Docker workspaces for LLM coding agents"
    license: "MIT"

release:
  github:
    owner: peasant-labs
    name: zone
```

**Key:** `homebrew_casks` (not `brews`) — `brews` was deprecated in GoReleaser v2.10.

**Key:** `version: 2` header is required to avoid warnings. GoReleaser v2 is the current major.

**Archive format:** `tar.gz` is the default for linux/darwin; `zip` override for windows is the standard convention.

### Pattern 4: GitHub Actions CI Workflow

Two workflows: `ci.yml` (every push/PR) and `release.yml` (on tag push).

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: ["**"]
  pull_request:
    branches: ["**"]

jobs:
  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - run: go build ./...
      - run: go test ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - uses: golangci/golangci-lint-action@v7
        with:
          version: v2.11

  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...

  goreleaser-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: "~> v2"
          args: release --snapshot --clean
```

**Important:** The CI check uses `goreleaser release --snapshot --clean`, not just `goreleaser check`. `goreleaser check` only validates the config syntax. `--snapshot` actually cross-compiles for all platforms, catching linker errors or missing source. This satisfies the CONTEXT.md requirement for a "GoReleaser snapshot build in CI to verify cross-compilation."

### Pattern 5: golangci-lint v2 Configuration

```yaml
# .golangci.yml
version: "2"

linters:
  default: standard
  enable:
    - errorlint      # error wrapping scheme (Go 1.13+)
    - errname        # sentinel errors prefixed with Err
    - bodyclose      # HTTP response body closed
    - contextcheck   # non-inherited context usage
    - gocritic       # style and correctness checks
    - misspell       # spelling in comments/strings
```

**Rationale for discretionary linter choices:**
- `default: standard` enables the safe baseline: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`
- Added `errorlint` because the spec mandates `%w` wrapping pattern throughout
- Added `errname` because the spec defines sentinel error naming (`ErrNoConfig`, etc.)
- `gocritic` gives broad style feedback without being too noisy
- `misspell` catches doc comment typos with no false positives

### Anti-Patterns to Avoid

- **Using `go:embed` in stub files:** `pkg/templates/templates.go` should have placeholder `//go:embed` directives that reference real template files. Create minimal empty `.tmpl` files so the embed compiles. If the templates are empty, the stub compiles and Phase 4 fills them in.
- **Skipping `fetch-depth: 0` in CI:** GoReleaser requires full git history to compute changelogs and version tags. Without it, `goreleaser` commands fail with "could not find tag" errors.
- **Using `brews:` in .goreleaser.yml:** Deprecated since v2.10. Use `homebrew_casks:`.
- **Forgetting `CGO_ENABLED=0`:** Cross-compilation to Darwin on a Linux runner requires pure-Go builds. Without this, `darwin/amd64` cross-compilation fails.
- **Empty `tests/` directory:** `go test ./...` must exit 0. Create stub `_test.go` files in `tests/` or the directory will be ignored but not cause failures. However, if any `_test.go` files have compile errors the entire suite fails — so test stubs must also compile cleanly.
- **`cmd/exec.go` naming conflict:** The package-level var name `execCmd` may conflict with Go's built-in `exec` package behavior. Use `execCmd` as the variable name to avoid any confusion; do not name it `exec` (would shadow the package path).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Subcommand routing | Custom flag parser | Cobra | Handles help text, usage, aliases, flag inheritance automatically |
| Cross-platform release | Custom CI matrix | GoReleaser | Handles archive naming, checksums, Homebrew, GitHub releases |
| Linting orchestration | Individual `go vet` calls | golangci-lint | Runs 50+ analyzers in parallel with caching |
| Vulnerability scanning | Manual CVE checks | govulncheck | Queries Go vuln database, understands call graphs |
| Version embedding | Manual `git describe` in Makefile | GoReleaser ldflags | Consistent version injection across all platforms |

**Key insight:** The entire value of this phase is getting the toolchain wired correctly. Every tool here (Cobra, GoReleaser, golangci-lint) has significant "incidental complexity" in initial configuration that the tool itself handles — the only work is writing the configuration correctly.

---

## Common Pitfalls

### Pitfall 1: Stub files that don't compile

**What goes wrong:** A stub `internal/docker/manager.go` that has `package docker` but no types causes no issues. But if any stub file imports a package that doesn't exist in `go.mod`, `go build ./...` fails.
**Why it happens:** Over-eager stubs that copy interface signatures from the spec without adding the required `go get` entries.
**How to avoid:** For Phase 1, stub files should contain only the package declaration and a doc comment. No type definitions, no imports except where required by Cobra registration in `cmd/`. Run `go build ./...` as the acceptance test.
**Warning signs:** `go: no required module provides package ...` errors.

### Pitfall 2: templates/ package fails to embed

**What goes wrong:** `pkg/templates/templates.go` uses `//go:embed *.tmpl` but no `.tmpl` files exist yet, causing a build error.
**Why it happens:** `//go:embed` with a glob pattern that matches zero files is a compile-time error in Go 1.16+.
**How to avoid:** Either: (a) create minimal empty `.tmpl` files as placeholders, or (b) defer the `//go:embed` directive to Phase 4 and use a simple `package templates` stub for now. Option (a) is cleaner since Phase 4 will fill them in.
**Warning signs:** `pattern *.tmpl: no matching files found` at compile time.

### Pitfall 3: GoReleaser snapshot requires GORELEASER_CURRENT_TAG

**What goes wrong:** `goreleaser release --snapshot` on a branch with no tags may warn or error about no previous tag found.
**Why it happens:** GoReleaser uses git history to infer version. In a fresh repo with no tags, it needs guidance.
**How to avoid:** In the CI snapshot job, set `GORELEASER_CURRENT_TAG=v0.0.0-dev` as an env var, or ensure the repo has at least one tag before running CI.
**Warning signs:** `couldn't find any tags` in goreleaser output.

### Pitfall 4: golangci-lint v1 config on v2 binary

**What goes wrong:** A `.golangci.yml` with `linters: enable-all: true` or `linters: disable-all: true` (v1 syntax) fails with the installed golangci-lint v2.11.4 — the v2 binary does not parse v1 config files.
**Why it happens:** Training data and tutorials still show v1 config syntax extensively.
**How to avoid:** Use `version: "2"` as the first line of `.golangci.yml` and use `linters.default: standard` (v2 syntax).
**Warning signs:** `golangci-lint: unknown command or invalid configuration syntax`.

### Pitfall 5: Makefile tab vs space indentation

**What goes wrong:** Makefiles require hard tabs for recipe lines. If an editor converts tabs to spaces, `make` fails with `Makefile:N: *** missing separator`.
**Why it happens:** Editors with "expand tabs" settings.
**How to avoid:** Write Makefile with explicit tabs. Verify with `cat -A Makefile` (tabs show as `^I`).

### Pitfall 6: `cmd/exec.go` variable shadowing

**What goes wrong:** Naming a file or variable `exec` can shadow the standard library `os/exec` package in future phases.
**Why it happens:** The `zone exec` command naturally maps to `exec.go` containing an `execCmd` variable.
**How to avoid:** Use `execCmd` for the variable name. The file `cmd/exec.go` is fine — file names don't shadow package names. The package declaration remains `package cmd`.

---

## Code Examples

Verified patterns from official sources:

### Cobra subcommand registration
```go
// Source: https://pkg.go.dev/github.com/spf13/cobra
// Minimal command with RunE that returns "not implemented"
var launchCmd = &cobra.Command{
    Use:     "launch",
    Aliases: []string{"up"},
    Short:   "Build (if needed) and run the container",
    Long:    `Launch builds the Docker image if needed and starts the container.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("not implemented")
    },
}
```

### GoReleaser version 2 header
```yaml
# Source: https://goreleaser.com/customization/
version: 2  # required in GoReleaser v2; omitting triggers a deprecation warning
```

### golangci-lint v2 minimal config
```yaml
# Source: https://golangci-lint.run/docs/configuration/file/
version: "2"
linters:
  default: standard
```

### go:embed with placeholder files
```go
// pkg/templates/templates.go
// Package templates provides embedded Dockerfile and script templates.
package templates

import "embed"

//go:embed *.tmpl
var FS embed.FS
```
The corresponding placeholder files (`Dockerfile.tmpl`, `entrypoint.sh.tmpl`, `zone-bashrc.tmpl`) must exist (even if empty) for this to compile.

### Makefile structure
```makefile
# All recipe lines MUST use tabs, not spaces
BINARY := zone
MODULE := github.com/peasant-labs/zone

.PHONY: build test lint fmt vet clean install

build:
	go build -o bin/$(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin/

install:
	go install .
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `brews:` in .goreleaser.yml | `homebrew_casks:` | GoReleaser v2.10 (2025) | Must use casks; brews deprecated |
| golangci-lint v1 config (`enable-all:`) | golangci-lint v2 config (`linters.default:`) | golangci-lint v2.0 (2025) | Config format is incompatible; v2 binary rejects v1 config |
| `goreleaser check` for CI | `goreleaser --snapshot --clean` | Always | `check` only validates config syntax; snapshot actually cross-compiles |
| Cobra v1.7 | Cobra v1.10.2 | Dec 2025 | Latest stable; v2.x not yet published to module proxy |

**Deprecated/outdated:**
- `brews:` in GoReleaser: deprecated v2.10, use `homebrew_casks:`
- golangci-lint v1 config syntax: not parsed by v2 binary; must migrate to `version: "2"`
- golangci-lint `enable-all: true`: removed in v2; use `default: all` instead

---

## Open Questions

1. **Homebrew tap write access in CI release workflow**
   - What we know: GoReleaser needs a `GITHUB_PERSONAL_AUTH_TOKEN` (or equivalent) with write access to `peasant-labs/homebrew-tap` to push the cask formula
   - What's unclear: Whether this secret is already configured in the repo, or needs to be created
   - Recommendation: The planner should note that the release workflow requires `HOMEBREW_TAP_TOKEN` secret in GitHub repo settings; document this as a manual setup step. The CI snapshot job does NOT need the token (snapshot does not publish).

2. **GORELEASER_CURRENT_TAG in CI on untagged commits**
   - What we know: `goreleaser release --snapshot` may warn on repos with no existing tags
   - What's unclear: Whether the fresh repo will have a tag before CI runs for the first time
   - Recommendation: Set `GORELEASER_CURRENT_TAG: v0.0.0-dev` as an env var on the goreleaser CI step for safety.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` package (built-in) |
| Config file | none — `go test` needs no config file |
| Quick run command | `go test ./...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DX-10 | GoReleaser config passes `goreleaser check` | smoke | `goreleaser check` | Wave 0 (config file created) |
| DX-10 | GoReleaser snapshot cross-compiles all targets | smoke | `goreleaser release --snapshot --clean` | Wave 0 (config file created) |
| (implicit) | `go build ./...` succeeds from clean checkout | smoke | `go build ./...` | Wave 0 (stub files created) |
| (implicit) | `go test ./...` exits 0 | smoke | `go test ./...` | Wave 0 (no test failures) |

> Note: Phase 1 has no unit tests of its own — the success criteria ARE the build/check commands. Tests in `tests/*.go` files are stubs that must compile cleanly but return no test functions yet.

### Sampling Rate
- **Per task commit:** `go build ./...`
- **Per wave merge:** `go build ./... && go test ./... && goreleaser check`
- **Phase gate:** `go build ./... && go test ./... && goreleaser check && goreleaser release --snapshot --clean` all green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `.goreleaser.yml` — must be created for `goreleaser check` to pass
- [ ] `.golangci.yml` — must be created for `golangci-lint run` to pass
- [ ] `go.mod` with `go 1.24` directive — created by `go mod init`
- [ ] All stub `.go` files — ~40 files as listed in spec Section 7
- [ ] `pkg/templates/*.tmpl` placeholder files — needed for `//go:embed` to compile
- [ ] `tests/*.go` compile-clean stubs — so `go test ./...` exits 0

---

## Sources

### Primary (HIGH confidence)
- Go module proxy (`go list -m -json`) — verified versions for cobra, bubbletea, lipgloss, toml, docker, vuln
- [GoReleaser CI/Actions docs](https://goreleaser.com/ci/actions/) — action versions, workflow structure
- [GoReleaser Homebrew Casks docs](https://goreleaser.com/customization/homebrew_casks/) — casks configuration
- [GoReleaser Homebrew Formulas deprecated](https://goreleaser.com/customization/homebrew_formulas/) — confirmed brews deprecated
- [golangci-lint config file docs](https://golangci-lint.run/docs/configuration/file/) — v2 config format
- `golangci-lint linters` command output — confirmed default enabled linters
- Spec Section 7 — authoritative project structure

### Secondary (MEDIUM confidence)
- [GoReleaser blog v2 announcement](https://goreleaser.com/blog/goreleaser-v2/) — v2 migration context
- WebSearch results for GoReleaser version — confirmed v2.14.3 as current (March 2026)

### Tertiary (LOW confidence)
- None — all critical claims verified from primary sources

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against Go module proxy (2026-03-26)
- Architecture: HIGH — directly from spec Section 7 + verified GoReleaser/golangci-lint docs
- Pitfalls: HIGH — based on verified tool behavior (golangci-lint v2 config incompatibility, GoReleaser embed rules, etc.)

**Research date:** 2026-03-26
**Valid until:** 2026-04-26 (stable tooling; GoReleaser releases frequently but v2 format is stable)
