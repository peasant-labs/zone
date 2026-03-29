# Phase 5: Harness Plugin System - Context

**Gathered:** 2026-03-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the harness plugin architecture: interface, BaseHarness, factory registry, fully-implemented claude-code harness, custom harness from config, and stub harnesses for opencode/gemini-cli/aider/codex-cli. Per-harness config validation ensures cross-harness keys produce specific errors. The `--prompt/-p` flag translates to harness-appropriate flags automatically. This phase does NOT launch containers (Phase 6) — it produces the harness objects that feed template data structs built in Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Claude Code harness (HAR-04)
- Install commands: `npm install -g @anthropic-ai/claude-code` (with `@version` suffix when `harness.version` is set, e.g., `@anthropic-ai/claude-code@1.0.26`)
- Entrypoint command: `claude` (bare command — extra args and prompt flag appended by entrypoint template)
- Health check: `claude --version`
- Home config dir: `~/.claude`
- Needs Node: true (default node_version "22")
- Required env vars: `ANTHROPIC_API_KEY`
- Default npm packages: none beyond claude-code itself (installed via HarnessInstallCommands, not NpmPackages)
- PromptFlag: `-p` (used for `--prompt` translation in HAR-10)
- skip_permissions handling (HAR-09): when `skip_permissions = true`, add `--dangerously-skip-permissions` to extra_args; defaults to false

### Stub harnesses (HAR-05)
- opencode, gemini-cli, aider, codex-cli each return descriptive "not yet implemented" error from Validate()
- Error message format per spec: `the "X" harness is not yet fully implemented; use harness = "custom" with install_commands and entrypoint_command to configure it manually`
- Stubs still implement the full Harness interface (Name(), etc.) — they just fail at Validate()

### Custom harness (HAR-06)
- `entrypoint_command` is required — Validate() returns error if empty: `custom harness requires "entrypoint_command" in [harness] config`
- `install_commands` is optional — an empty list is valid (tool may be pre-installed in base image)
- `config_dirs` passed through to EntrypointData.ConfigCopyCommands as copy-on-start commands (Phase 7 mounts the actual host dirs)
- `required_env` stored for pre-launch validation (Phase 7 checks these before Docker build)
- `health_check` optional — if set, populates DockerfileData.HealthCheck
- `aliases` and `shell_rc` populate ShellRCData via the standard harness interface methods
- NeedsNode/NeedsPython: false by default for custom harness (user installs runtimes via install_commands)

### Per-harness config validation (HAR-07)
- Each harness's Validate() checks that only its supported fields are set in HarnessConfig
- Cross-harness key produces specific error: `harness "X" does not support key "Y" (that key is specific to "Z")`
- Common fields (version, extra_args) are allowed on all harnesses
- Custom harness allows all custom-specific fields plus common fields
- Validation runs in `harness.Get()` after factory construction — before any template rendering

### Prompt flag translation (HAR-10)
- Each harness implements `PromptFlag() string` returning the flag name (e.g., `-p` for claude-code, `--message` for aider)
- When `zone launch -- -p "task"` is used, the prompt flag is prepended to extra args
- If a harness returns empty PromptFlag() and user provides --prompt, error: `harness "X" does not support the --prompt flag`
- Stub harnesses return empty PromptFlag() (they fail at Validate() before this matters)

### Harness→template integration
- A `BuildDockerfileData(h Harness, cfg *MergedConfig)` function in `internal/docker/` takes a Harness and MergedConfig, calls all harness methods, and returns a populated DockerfileData
- Similarly `BuildEntrypointData(h Harness, cfg *MergedConfig)` and `BuildShellRCData(h Harness, cfg *MergedConfig)`
- These are the single integration point between harness system and template rendering
- Lives in `internal/docker/` per import graph: `internal/docker -> internal/harness OK`

### Claude's Discretion
- Internal struct layout of each concrete harness (field names, constructor pattern)
- Test strategy (table-driven vs individual test functions)
- Whether to use a `supportedKeys` map or reflection for cross-harness validation
- Exact error wrapping patterns within Validate() methods

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Harness architecture
- `zone-spec.md` §10 (lines 823-944) — Complete Harness interface, BaseHarness defaults, registry pattern, typed HarnessConfig, stub behavior, Validate() error format
- `zone-spec.md` §5 (lines 543-594) — Harness-specific config keys by type, security note on skip_permissions, per-harness validation error format

### Template integration
- `zone-spec.md` §11 (lines 947-1148) — Dockerfile.tmpl, entrypoint.sh.tmpl, zone-bashrc.tmpl showing all harness-populated fields
- `zone-spec.md` §12 (lines 1150-1237) — Docker Manager key responsibilities (harness consumed here in Phase 6)

### Config types
- `zone-spec.md` §4 (lines 254-458) — Config struct definitions, env var forwarding examples with ANTHROPIC_API_KEY
- `internal/config/harness_config.go` — Typed HarnessConfig struct (already implemented in Phase 2)
- `internal/config/types.go` — MergedConfig, RepoConfig, GlobalConfig with HarnessConfig field

### Existing template data structs (Phase 4)
- `internal/docker/dockerfile.go` — DockerfileData struct with HarnessInstallCommands, HealthCheck, NeedsNode, etc.
- `internal/docker/entrypoint.go` — EntrypointData struct with EntrypointCommand, ConfigCopyCommands
- `internal/docker/shellrc.go` — ShellRCData struct with Aliases, ShellRC, WelcomeMessage

### Import graph
- `zone-spec.md` §7 (lines 717-727) — Enforced import rules: `internal/docker -> internal/harness OK`

### Requirements
- `.planning/REQUIREMENTS.md` — HAR-01 through HAR-10

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/harness_config.go` — HarnessConfig struct fully defined with all harness-specific fields (Phase 2)
- `internal/config/types.go` — MergedConfig with Harness field, DefaultGlobalConfig/DefaultRepoConfig with defaults
- `internal/docker/dockerfile.go` — DockerfileData struct and RenderDockerfile() ready to consume harness data
- `internal/docker/entrypoint.go` — EntrypointData struct and RenderEntrypoint() ready to consume harness data
- `internal/docker/shellrc.go` — ShellRCData struct and RenderShellRC() ready to consume harness data
- `internal/docker/platform.go` — HostUID(), MacOSUsername(), DetectGitIdentity() for populating template data

### Established Patterns
- Stub-first: all harness files exist as package-only stubs (`internal/harness/*.go`)
- Error wrapping: `fmt.Errorf("context: %w", err)` throughout codebase
- Config merge: `*bool` for nullable booleans enabling merge semantics (used by skip_permissions)
- Typed structs over maps: HarnessConfig is a typed struct, not map[string]interface{} (HAR-08 already satisfied by Phase 2)

### Integration Points
- `internal/harness/` stubs → fully implement Harness interface and registry
- `internal/docker/` render functions → new builder functions bridge harness→template data
- `internal/config/` HarnessConfig → consumed by harness constructors via registry factory functions
- `cmd/` commands → Phase 6/8 will call `harness.Get()` from command handlers

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. All decisions auto-selected from recommended defaults following the spec's prescriptive patterns. The spec provides exact Go code for the interface, BaseHarness, registry, and stub behavior.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 05-harness-plugin-system*
*Context gathered: 2026-03-29*
