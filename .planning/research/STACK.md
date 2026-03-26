# Stack Research

**Domain:** Go CLI tool — Docker workspace manager for LLM coding agents
**Researched:** 2026-03-26
**Confidence:** HIGH (core stack pre-decided in spec, versions verified against pkg.go.dev and official release pages)

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.23+ | Primary language | Standard for CLI tooling; native concurrency for container lifecycle; single binary distribution; required by all libraries below |
| github.com/spf13/cobra | v1.10.2 | CLI command framework (14 commands) | De facto standard for Go CLIs; used by kubectl, docker CLI, gh. Provides subcommand routing, flag parsing, shell completions, help generation. No viable competitor at this maturity level |
| charm.land/bubbletea/v2 | v2.0.0 | Interactive TUI framework | Elm-architecture model makes complex TUI state manageable. v2.0.0 (Feb 23, 2026) is the first stable v2 — ships the "Cursed Renderer" (10x faster, ncurses-based), progressive keyboard enhancement, native clipboard. Import path moved to charm.land vanity domain |
| charm.land/lipgloss/v2 | v2.0.0 | Terminal styling (colors, borders, layout) | Companion to BubbleTea. CSS-inspired API. v2 released same day as BubbleTea v2; now "pure" — BubbleTea manages I/O and directs Lip Gloss. Must match bubbletea major version |
| charm.land/bubbles/v2 | v2.0.0 | Pre-built TUI components (spinner, progress bar, list, viewport, text input) | Saves significant TUI implementation work. Provides spinner (build progress), viewport (log viewer), list (workspace picker), textinput (init wizard). Must match bubbletea major version |
| github.com/BurntSushi/toml | v1.5.0 | TOML config parsing with strict decoding | Supports `toml.DecodeStrict()` which rejects unknown keys — essential for catching typos in zone.toml. Alternative pelletier/go-toml lacks the strict-mode ergonomics for edit-distance suggestions. Most widely imported Go TOML library (37k+ dependents) |
| github.com/docker/docker/client | v28.5.2+incompatible | Docker Engine API client | Official SDK used by docker CLI itself. Provides container lifecycle (create, start, stop, remove), image management (build, pull), network management, exec, attach, log streaming. The `+incompatible` suffix is cosmetic — library is production-stable |
| github.com/coreos/go-iptables | v0.8.0 | Host-side iptables rule management for network sandboxing | Simple Go wrapper around the `iptables` CLI binary. Correct tool for the Zone v1 approach (exec iptables, not raw netlink). Linux-only — matches Zone's v1 network filtering scope. Maintained by CoreOS/Red Hat |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/gofrs/flock | v0.13.0 | File locking for .zone/ directory | Prevents concurrent `zone launch` invocations from corrupting the .zone/container-id and .zone/config-hash files. Use for all writes to .zone/ cache directory |
| github.com/agnivade/levenshtein | v1.2.0 | Edit-distance for config key suggestions | Powers "did you mean 'entrypoint_command'?" suggestions when BurntSushi/toml rejects unknown keys. Minimal, fast, no dependencies |
| Standard library: text/template + embed | stdlib | Dockerfile/entrypoint/RC template rendering | Go's built-in template engine is sufficient for Dockerfile generation. Use `//go:embed templates/*` to bundle template files in the binary — no runtime file dependency |
| Standard library: encoding/json | stdlib | --json output for scriptability | Zone requires JSON output on status, ls, config, logs commands. No external dependency needed |
| Standard library: os/exec | stdlib | SSH agent forwarding, hook execution, iptables invocation | Preferred over shelling out via shell string. Use for pre_build/post_stop hooks and coreos/go-iptables fallback paths |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| GoReleaser v2.14.3 | Binary distribution — builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64; generates Homebrew tap formula | Configure `.goreleaser.yaml` at repo root. Use `goreleaser build --snapshot` for local testing. The spec calls for go install + Homebrew tap + prebuilt binaries — GoReleaser handles all three |
| golangci-lint | Static analysis and linting | Run in CI. Catches shadowed variables (common Go gotcha), unchecked errors, and context misuse which are real risks in container lifecycle code |
| go test -race | Race detector | Critical for testing container lifecycle state (image ID cache, container ID cache, file locks). Always enable in CI |
| Docker (daemon) | Integration testing target | Tests need a real Docker daemon. Use `testcontainers-go` or a raw Docker socket in CI |

## Installation

```bash
# Initialize module
go mod init zone

# Core CLI framework
go get github.com/spf13/cobra@v1.10.2

# TUI stack (all three must be v2 together)
go get charm.land/bubbletea/v2@v2.0.0
go get charm.land/lipgloss/v2@v2.0.0
go get charm.land/bubbles/v2@v2.0.0

# Config parsing
go get github.com/BurntSushi/toml@v1.5.0

# Docker SDK
go get github.com/docker/docker/client@v28.5.2+incompatible
go get github.com/docker/docker/api/types@v28.5.2+incompatible

# Network filtering (Linux-only, build-tag guarded)
go get github.com/coreos/go-iptables@v0.8.0

# File locking
go get github.com/gofrs/flock@v0.13.0

# Edit distance for config suggestions
go get github.com/agnivade/levenshtein@v1.2.0

# Dev tooling (not in go.mod)
go install github.com/goreleaser/goreleaser/v2@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| charm.land/bubbletea/v2 | bubbletea v1 (github.com/charmbracelet/bubbletea) | Only if you need to avoid breaking API changes from v1→v2 migration. v2 is stable as of Feb 2026 and has significant renderer improvements — prefer v2 for new projects |
| github.com/BurntSushi/toml | github.com/spf13/viper | Use Viper if you need multi-format config (YAML, JSON, env vars, remote config). Zone deliberately wants TOML-only with strict decoding — Viper's flexibility adds complexity without benefit here |
| github.com/BurntSushi/toml | github.com/pelletier/go-toml | go-toml v2 is also good. Choose BurntSushi if strict decoding with unknown-key errors is a primary requirement; choose pelletier for richer query/path API |
| github.com/coreos/go-iptables | google/nftables | Use google/nftables if targeting Docker Engine 29+ (which defaults nftables) or modern Debian/Ubuntu where iptables is nftables-backed. Zone v1 targets iptables for simplicity; v2 should evaluate nftables |
| github.com/docker/docker/client | github.com/docker/go-sdk | docker/go-sdk is a newer higher-level client. Still immature (no stable release as of early 2026). Use docker/docker/client for now — it is what the docker CLI itself uses |
| github.com/agnivade/levenshtein | cobra's built-in suggestion | Cobra provides `SuggestionsFor()` for command name suggestions. For config key suggestions (TOML keys, not commands), a standalone levenshtein library is required |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| github.com/charmbracelet/bubbletea (v1 import path) | v2.0.0 is stable as of Feb 2026 with a new vanity import path `charm.land/bubbletea/v2`. Starting a new project on v1 means immediate migration debt | charm.land/bubbletea/v2 |
| github.com/spf13/viper | Overkill for Zone's two-file TOML config. Adds heavy dependency tree (remote config, many format parsers) and loses strict-decoding ergonomics. BurntSushi/toml's `DecodeStrict` is the right primitive | github.com/BurntSushi/toml |
| os.Exec shell string ("iptables -A ...") | Shell injection risk, no error classification. Use coreos/go-iptables which wraps iptables properly with typed errors | github.com/coreos/go-iptables |
| fsouza/go-dockerclient | Third-party Docker client that lags behind the official API. Historically popular but the official docker/docker client is now well-documented and used internally | github.com/docker/docker/client |
| Direct netlink for iptables | Requires CAP_NET_ADMIN which conflicts with Zone's security model (host-side enforcement without container privilege). Also far more complex than coreos/go-iptables | github.com/coreos/go-iptables |
| google/nftables in v1 | Docker's nftables backend is experimental as of Docker 29. iptables is still the default. Adding nftables support in v1 adds compatibility surface with no clear benefit | Defer to v2 |

## Stack Patterns by Variant

**If running on macOS (no network filtering):**
- Build-tag guard the coreos/go-iptables import behind `//go:build linux`
- The `NetworkManager` should be a no-op stub on darwin
- All other stack components work cross-platform

**If running in a non-TTY environment (CI, piped output):**
- Skip BubbleTea program initialization entirely
- Fall back to plain fmt.Println with structured text
- Detect with `term.IsTerminal(int(os.Stdout.Fd()))` from golang.org/x/term
- The `--plain` flag forces this path even in a TTY

**If building the init wizard (interactive config generation):**
- Use charm.land/bubbles/v2 `textinput` for field entry
- Use charm.land/bubbles/v2 `list` for harness selection
- Compose these as BubbleTea sub-models, not separate programs

**If streaming build logs:**
- Use charm.land/bubbles/v2 `viewport` for scrollable log view
- Pipe docker build output through a channel to the BubbleTea Update loop via `tea.Cmd`
- Do NOT block the BubbleTea event loop with synchronous Docker API reads

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| charm.land/bubbletea/v2@v2.0.0 | charm.land/lipgloss/v2@v2.0.0, charm.land/bubbles/v2@v2.0.0 | All three v2 libraries released together Feb 23, 2026. Must upgrade all three together. v1 and v2 cannot mix in the same program |
| github.com/docker/docker/client@v28.5.2 | Docker Engine 24+ | The `+incompatible` suffix means the module predates Go modules but is otherwise stable. Requires Docker Engine API version negotiation (handled automatically by the client) |
| github.com/spf13/cobra@v1.10.2 | github.com/spf13/pflag (auto-pulled) | Cobra manages its own pflag dependency. Do not import pflag directly unless overriding Cobra internals |
| github.com/BurntSushi/toml@v1.5.0 | Go 1.18+ | Requires generics-era Go. Compatible with all other stack components |

## Sources

- https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0 — BubbleTea v2.0.0 release date (Feb 24, 2025 tag; stable release Feb 23, 2026), import path, breaking changes. MEDIUM confidence (WebFetch)
- https://charm.land/blog/v2/ — Official Charm announcement of v2.0.0 stable for BubbleTea, Lip Gloss, Bubbles (Feb 23, 2026). HIGH confidence (official source)
- https://pkg.go.dev/github.com/docker/docker/client?tab=versions — Docker SDK latest v28.5.2+incompatible, published Nov 5, 2025. HIGH confidence (pkg.go.dev)
- https://github.com/spf13/cobra/releases — Cobra v1.10.2 latest release, Dec 2025. HIGH confidence (WebFetch)
- https://github.com/BurntSushi/toml/tree/v1.5.0 — BurntSushi/toml v1.5.0, Dec 2025. HIGH confidence (WebSearch + pkg.go.dev)
- https://pkg.go.dev/github.com/coreos/go-iptables/iptables — v0.8.0, Aug 2024. MEDIUM confidence (pkg.go.dev, may not be latest)
- https://pkg.go.dev/github.com/gofrs/flock — v0.13.0, Oct 2025. MEDIUM confidence (pkg.go.dev)
- https://goreleaser.com/blog/goreleaser-v2.14/ — GoReleaser v2.14.3, Mar 9, 2026. HIGH confidence (official blog)
- https://docs.docker.com/engine/network/firewall-nftables/ — Docker nftables experimental in Engine 29+. HIGH confidence (official Docker docs)

---
*Stack research for: Zone — Go CLI Docker workspace manager for LLM coding agents*
*Researched: 2026-03-26*
