# Phase 1: Project Scaffold - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Go module initialization, full package skeleton with stub files, CI pipeline, GoReleaser config, and Makefile. Users can clone the repo, run `go build ./...` successfully, and CI passes on every commit. No functional code — just the structure that all subsequent phases build on.

</domain>

<decisions>
## Implementation Decisions

### Module path & hosting
- Go module path: `github.com/peasant-labs/zone`
- Repo already exists at github.com/peasant-labs/zone
- Minimum Go version: 1.24

### CI pipeline
- GitHub Actions on push and PR
- Linux-only runners (ubuntu-latest) — no macOS CI runners
- Checks: `go build ./...`, `go test ./...`, `golangci-lint`, `goreleaser check`, `govulncheck`
- GoReleaser snapshot build in CI to verify cross-compilation for all platforms

### Scaffold depth
- Create ALL directories and packages from spec Section 7 with stub files
- Each stub file has package declaration and doc comment
- All cmd/* files register Cobra subcommands with placeholder `RunE` returning "not implemented" — `zone --help` shows all commands from day one
- Makefile targets: build, test, lint, fmt, vet, clean, install

### GoReleaser targets
- Platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Homebrew tap configured (peasant-labs/homebrew-tap repo)
- No Docker image of the CLI — zone runs on the host
- Archive format: Claude's discretion (tar.gz for linux/darwin, zip for windows is standard)

### Claude's Discretion
- golangci-lint configuration (which linters to enable)
- Archive format details (tar.gz vs zip per platform)
- Exact Makefile target implementations
- Stub file doc comments and placeholder text
- CI workflow naming and job structure

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure & conventions
- `zone-spec.md` §7 — Full project directory tree, all package paths, import graph rules (enforced)
- `zone-spec.md` §8 — Error handling convention (sentinel errors, wrapping pattern, exit code mapping)

### Distribution
- `.planning/PROJECT.md` — Core value, tech stack constraints, key decisions
- `.planning/REQUIREMENTS.md` — DX-10 (GoReleaser configuration for binary distribution)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project, only LICENSE and zone-spec.md exist

### Established Patterns
- None yet — this phase establishes the foundational patterns

### Integration Points
- go.mod initializes the module that all subsequent phases import from
- cmd/root.go establishes the Cobra root command that all phases add subcommands to
- CI pipeline validates every subsequent phase's code on push
- .goreleaser.yml must stay valid as dependencies are added in later phases

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for Go project scaffolding.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-project-scaffold*
*Context gathered: 2026-03-26*
