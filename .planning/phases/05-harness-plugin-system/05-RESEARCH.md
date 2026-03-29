# Phase 5: Harness Plugin System - Research

**Researched:** 2026-03-29
**Domain:** Go plugin interface pattern, harness registry, template bridge functions
**Confidence:** HIGH

## Summary

Phase 5 implements the harness plugin system entirely within the existing Go codebase.
All dependencies are already installed (stdlib only — no new packages needed). The
`internal/harness/` package already has stub files for every harness; this phase
fills them in. The `internal/docker/` package already has the three `*Data` structs
that consume harness output; this phase adds builder functions that bridge harness
objects to those structs.

The work is purely additive Go: define the `Harness` interface and `BaseHarness`
defaults in `harness.go`, fill in each harness file, wire the registry + `Get()`
function, and add three `Build*Data` functions in `internal/docker/`. No new external
dependencies are introduced — all patterns are prescribed verbatim in `zone-spec.md`
§10, so there is no design ambiguity to resolve.

**Primary recommendation:** Follow the spec exactly — the interface definition,
`BaseHarness` embedding pattern, registry map, stub error format, and per-harness
field validation are all specified with exact Go code. Deviate only in the internal
struct layout and constructor pattern (Claude's Discretion).

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Claude Code harness (HAR-04)
- Install commands: `npm install -g @anthropic-ai/claude-code` (with `@version` suffix
  when `harness.version` is set, e.g., `@anthropic-ai/claude-code@1.0.26`)
- Entrypoint command: `claude` (bare command — extra args and prompt flag appended by
  entrypoint template)
- Health check: `claude --version`
- Home config dir: `~/.claude`
- Needs Node: true (default node_version "22")
- Required env vars: `ANTHROPIC_API_KEY`
- Default npm packages: none beyond claude-code itself (installed via
  HarnessInstallCommands, not NpmPackages)
- PromptFlag: `-p` (used for `--prompt` translation in HAR-10)
- skip_permissions handling (HAR-09): when `skip_permissions = true`, add
  `--dangerously-skip-permissions` to extra_args; defaults to false

#### Stub harnesses (HAR-05)
- opencode, gemini-cli, aider, codex-cli each return descriptive "not yet implemented"
  error from `Validate()`
- Error message format: `the "X" harness is not yet fully implemented; use harness =
  "custom" with install_commands and entrypoint_command to configure it manually`
- Stubs still implement the full Harness interface (Name(), etc.) — they just fail at
  Validate()

#### Custom harness (HAR-06)
- `entrypoint_command` is required — Validate() returns error if empty: `custom harness
  requires "entrypoint_command" in [harness] config`
- `install_commands` is optional — an empty list is valid
- `config_dirs` passed through to EntrypointData.ConfigCopyCommands as copy-on-start
  commands (Phase 7 mounts the actual host dirs)
- `required_env` stored for pre-launch validation (Phase 7 checks these before Docker
  build)
- `health_check` optional — if set, populates DockerfileData.HealthCheck
- `aliases` and `shell_rc` populate ShellRCData via the standard harness interface
  methods
- NeedsNode/NeedsPython: false by default for custom harness

#### Per-harness config validation (HAR-07)
- Each harness's Validate() checks that only its supported fields are set in
  HarnessConfig
- Cross-harness key produces specific error: `harness "X" does not support key "Y"
  (that key is specific to "Z")`
- Common fields (version, extra_args) are allowed on all harnesses
- Custom harness allows all custom-specific fields plus common fields
- Validation runs in `harness.Get()` after factory construction — before any template
  rendering

#### Prompt flag translation (HAR-10)
- Each harness implements `PromptFlag() string` returning the flag name (e.g., `-p` for
  claude-code, `--message` for aider)
- When `zone launch -- -p "task"` is used, the prompt flag is prepended to extra args
- If a harness returns empty PromptFlag() and user provides --prompt, error: `harness
  "X" does not support the --prompt flag`
- Stub harnesses return empty PromptFlag() (they fail at Validate() before this matters)

#### Harness→template integration
- `BuildDockerfileData(h Harness, cfg *MergedConfig)` in `internal/docker/` — calls all
  harness methods, returns populated DockerfileData
- `BuildEntrypointData(h Harness, cfg *MergedConfig)` — returns populated EntrypointData
- `BuildShellRCData(h Harness, cfg *MergedConfig)` — returns populated ShellRCData
- These are the single integration point between harness system and template rendering
- Lives in `internal/docker/` per import graph: `internal/docker -> internal/harness OK`

### Claude's Discretion
- Internal struct layout of each concrete harness (field names, constructor pattern)
- Test strategy (table-driven vs individual test functions)
- Whether to use a `supportedKeys` map or reflection for cross-harness validation
- Exact error wrapping patterns within Validate() methods

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HAR-01 | Harness interface defines identity, installation, runtime, dependencies, shell, lifecycle methods | Spec §10 provides exact Go interface definition — copy verbatim |
| HAR-02 | BaseHarness provides default implementations for optional methods | Spec §10 provides exact BaseHarness struct with 9 default methods |
| HAR-03 | Factory registry maps harness names to constructors | Spec §10 provides exact registry map + Get() function pattern |
| HAR-04 | `claude-code` harness fully implemented with install, health check, env vars, config dir | CONTEXT.md locks all field values; NeedsNode=true, NodeVersion default "22" |
| HAR-05 | `opencode`, `gemini-cli`, `aider`, `codex-cli` return descriptive "not yet implemented" errors | Spec §10 provides exact Validate() error string; stub files already exist |
| HAR-06 | `custom` harness supports install_commands, entrypoint_command, config_dirs, required_env, health_check, aliases, shell_rc | CONTEXT.md locks behavior; entrypoint_command required, rest optional |
| HAR-07 | Each harness validates only its supported config keys; cross-harness keys produce specific errors | Error format locked; use supportedKeys set in each Validate() |
| HAR-08 | HarnessConfig is a typed struct (not map[string]interface{}) | Already satisfied by Phase 2 — HarnessConfig fully defined in internal/config/harness_config.go |
| HAR-09 | `skip_permissions` for claude-code defaults to false | Locked: when true, add `--dangerously-skip-permissions` to extra_args |
| HAR-10 | `--prompt`/`-p` flag translates to harness-appropriate prompt flag automatically | Locked: PromptFlag() method on interface; error when harness returns "" |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`fmt`, `errors`) | go 1.25.5 | Interface definition, error wrapping | No new dependencies needed |

### Supporting
No new external packages are required. All patterns use stdlib only.

**Existing packages already in go.mod (used by this phase):**
- `github.com/stretchr/testify v1.10.0` — test assertions (already present)
- `github.com/peasant-labs/zone/internal/config` — HarnessConfig, MergedConfig types

**Installation:** No new `go get` needed. All imports are stdlib or already in go.mod.

## Architecture Patterns

### Recommended File Layout

```
internal/harness/
├── harness.go         # Harness interface + BaseHarness + registry + Get()
├── claude_code.go     # ClaudeCode struct — fully implemented
├── custom.go          # Custom struct — fully implemented
├── opencode.go        # OpenCode stub — Validate() returns "not yet implemented"
├── gemini_cli.go      # GeminiCLI stub — Validate() returns "not yet implemented"
├── aider.go           # Aider stub — Validate() returns "not yet implemented"
└── codex_cli.go       # CodexCLI stub — Validate() returns "not yet implemented"

internal/docker/
├── harness_bridge.go  # BuildDockerfileData, BuildEntrypointData, BuildShellRCData
│                      # (NEW file — single integration point between harness and templates)
└── [existing files unchanged]
```

### Pattern 1: Harness Interface + BaseHarness Embedding

**What:** Define a single `Harness` interface with 20 methods spanning identity,
installation, runtime, dependencies, shell, and lifecycle. `BaseHarness` is an empty
struct that provides no-op defaults for optional methods. Concrete harnesses embed
`BaseHarness` and override only what they implement.

**When to use:** For every concrete harness type.

```go
// Source: zone-spec.md §10 (lines 830-878)
package harness

import (
    "fmt"
    "github.com/peasant-labs/zone/internal/config"
)

type Harness interface {
    Name() string
    Version() string
    InstallCommands() []string
    PostInstallCommands() []string
    HealthCheck() string
    EntrypointCommand() string
    PromptFlag() string
    RequiredEnvVars() []string
    HomeConfigDir() string
    ExtraConfigDirs() []string
    DefaultAptPackages() []string
    DefaultNpmPackages() []string
    DefaultPipPackages() []string
    NeedsNode() bool
    NeedsPython() bool
    ShellRC() []string
    Aliases() map[string]string
    WelcomeMessage() string
    NodeVersion() string   // NOTE: not in spec interface — see Pattern 2 below
    Validate() error
}

type BaseHarness struct{}

func (b BaseHarness) Version() string              { return "" }
func (b BaseHarness) PostInstallCommands() []string { return nil }
func (b BaseHarness) HealthCheck() string           { return "" }
func (b BaseHarness) PromptFlag() string            { return "" }
func (b BaseHarness) ExtraConfigDirs() []string     { return nil }
func (b BaseHarness) ShellRC() []string             { return nil }
func (b BaseHarness) Aliases() map[string]string    { return nil }
func (b BaseHarness) WelcomeMessage() string        { return "" }
func (b BaseHarness) Validate() error               { return nil }
```

**Important:** The spec interface does NOT include `NodeVersion()` or `PythonVersion()`
as interface methods. `NeedsNode() bool` and `NeedsPython() bool` are interface methods.
The version strings are passed separately to `BuildDockerfileData` via MergedConfig
(which carries `HarnessConfig.NodeVersion`). See Pattern 3 for how the bridge function
handles defaults.

### Pattern 2: ClaudeCode Harness Implementation

**What:** Concrete struct embedding BaseHarness, storing `*config.HarnessConfig`.
Validate() performs two checks: (1) cross-harness key rejection, (2) skip_permissions
handling.

**When to use:** Only for `claude_code.go`.

```go
// Source: zone-spec.md §10 + CONTEXT.md locked decisions
type ClaudeCode struct {
    BaseHarness
    config *config.HarnessConfig
}

func (c *ClaudeCode) Name() string { return "claude-code" }

func (c *ClaudeCode) InstallCommands() []string {
    pkg := "@anthropic-ai/claude-code"
    if c.config.Version != "" {
        pkg += "@" + c.config.Version
    }
    return []string{"npm install -g " + pkg}
}

func (c *ClaudeCode) HealthCheck() string      { return "claude --version" }
func (c *ClaudeCode) EntrypointCommand() string { return "claude" }
func (c *ClaudeCode) PromptFlag() string        { return "-p" }
func (c *ClaudeCode) RequiredEnvVars() []string { return []string{"ANTHROPIC_API_KEY"} }
func (c *ClaudeCode) HomeConfigDir() string     { return "~/.claude" }
func (c *ClaudeCode) NeedsNode() bool           { return true }
func (c *ClaudeCode) NeedsPython() bool         { return false }
func (c *ClaudeCode) DefaultAptPackages() []string { return nil }
func (c *ClaudeCode) DefaultNpmPackages() []string { return nil }
func (c *ClaudeCode) DefaultPipPackages() []string { return nil }

func (c *ClaudeCode) Validate() error {
    // Reject cross-harness keys
    if c.config.PythonVersion != "" {
        return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
            "claude-code", "python_version", "aider")
    }
    if len(c.config.InstallCommands) > 0 {
        return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
            "claude-code", "install_commands", "custom")
    }
    // ... similar checks for entrypoint_command, config_dirs, required_env,
    //     health_check, aliases, shell_rc
    return nil
}
```

### Pattern 3: Registry + Get()

**What:** A package-level map of factory functions. `Get()` constructs the harness
and immediately calls `Validate()`.

```go
// Source: zone-spec.md §10 (lines 884-904)
var registry = map[string]func(*config.HarnessConfig) Harness{
    "claude-code": func(c *config.HarnessConfig) Harness { return &ClaudeCode{config: c} },
    "opencode":    func(c *config.HarnessConfig) Harness { return &OpenCode{config: c} },
    "gemini-cli":  func(c *config.HarnessConfig) Harness { return &GeminiCLI{config: c} },
    "aider":       func(c *config.HarnessConfig) Harness { return &Aider{config: c} },
    "codex-cli":   func(c *config.HarnessConfig) Harness { return &CodexCLI{config: c} },
    "custom":      func(c *config.HarnessConfig) Harness { return &Custom{config: c} },
}

func Get(name string, cfg *config.HarnessConfig) (Harness, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown harness %q, available: %v", name, availableNames())
    }
    h := factory(cfg)
    if err := h.Validate(); err != nil {
        return nil, fmt.Errorf("harness %q config: %w", name, err)
    }
    return h, nil
}

func availableNames() []string {
    names := make([]string, 0, len(registry))
    for k := range registry {
        names = append(names, k)
    }
    sort.Strings(names)
    return names
}
```

### Pattern 4: Stub Harnesses

**What:** Minimal struct that implements the interface only to satisfy the compiler.
`Validate()` always returns the spec-prescribed error. All other methods delegate to
`BaseHarness` or return sensible zero values.

```go
// Source: zone-spec.md §10 (lines 936-943)
type OpenCode struct {
    BaseHarness
    config *config.HarnessConfig
}

func (o *OpenCode) Name() string { return "opencode" }
// DefaultAptPackages, DefaultNpmPackages, DefaultPipPackages, RequiredEnvVars,
// HomeConfigDir, EntrypointCommand, InstallCommands all return zero values via BaseHarness
// or explicit "" / nil returns.

func (o *OpenCode) Validate() error {
    return fmt.Errorf(
        "the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually",
        o.Name(),
    )
}
```

### Pattern 5: Harness Bridge Functions (new file `internal/docker/harness_bridge.go`)

**What:** Three pure functions in `internal/docker/` that accept a `Harness` and
`*config.MergedConfig` and return the corresponding template data struct. These are the
ONLY code that should call harness methods — callers (Phase 6 cmd handlers) use these
rather than calling harness methods directly.

```go
// Source: CONTEXT.md locked decisions on harness→template integration
package docker

import (
    "github.com/peasant-labs/zone/internal/config"
    "github.com/peasant-labs/zone/internal/harness"
)

func BuildDockerfileData(h harness.Harness, cfg *config.MergedConfig) DockerfileData {
    nodeVer := cfg.Harness.NodeVersion
    if nodeVer == "" {
        nodeVer = "22" // default per spec §5
    }
    pythonVer := cfg.Harness.PythonVersion
    if pythonVer == "" {
        pythonVer = "3.12" // default per spec §5
    }
    return DockerfileData{
        BaseImage:              cfg.Zone.BaseImage,
        HostUID:                0, // caller populates via HostUID()
        AptPackages:            merge(cfg.Packages.Apt, h.DefaultAptPackages()),
        NeedsNode:              h.NeedsNode(),
        NodeVersion:            nodeVer,
        NeedsPython:            h.NeedsPython(),
        PythonVersion:          pythonVer,
        NpmPackages:            merge(cfg.Packages.Npm, h.DefaultNpmPackages()),
        PipPackages:            merge(cfg.Packages.Pip, h.DefaultPipPackages()),
        HarnessInstallCommands: h.InstallCommands(),
        HealthCheck:            h.HealthCheck(),
        Shell:                  cfg.Zone.Shell,
        MountPath:              cfg.Workspace.MountPath,
        // MacOSUsername, HostUID, InstallZsh, PostInstallCommands populated by caller
    }
}
```

**Note on HostUID:** The bridge function receives MergedConfig which does not contain
HostUID (it is a runtime value, not a config value). The caller (Phase 6) must set
`DockerfileData.HostUID` and `DockerfileData.MacOSUsername` after calling
`BuildDockerfileData`. The bridge function populates everything derivable from harness
methods and MergedConfig.

### Pattern 6: Per-Harness Key Validation

**What:** Each `Validate()` method checks that fields belonging to OTHER harnesses are
unset. Use a direct field-by-field check (not reflection) — it is explicit, readable,
and produces accurate "specific to X" error messages.

**Recommended approach:** Define which fields each harness owns. Check all
non-owning harnesses' specific fields are zero/nil.

```go
// Claude-code Validate() checks:
//   - PythonVersion == ""         (aider-specific)
//   - InstallCommands == nil/[]    (custom-specific)
//   - EntrypointCommand == ""      (custom-specific)
//   - ConfigDirs == nil/[]         (custom-specific)
//   - RequiredEnv == nil/[]        (custom-specific — overlaps with RequiredEnvVars())
//   - CustomHealthCheck == ""      (custom-specific)
//   - CustomAliases == nil/{}      (custom-specific)
//   - CustomShellRC == nil/[]      (custom-specific)
//
// Stubs (opencode, gemini-cli, codex-cli) check:
//   - PythonVersion == ""          (aider-specific)
//   - [all custom fields empty]
//
// Aider Validate() checks (stub):
//   - SkipPermissions == nil       (claude-code-specific)
//   - [all custom fields empty]
//
// Custom Validate():
//   - EntrypointCommand != ""      (required)
//   - SkipPermissions == nil       (claude-code-specific — reject)
//   Note: custom allows all its own fields
```

**Key insight:** The `HarnessConfig` already contains `RequiredEnv []string` (for
custom harness). The `ClaudeCode.RequiredEnvVars()` interface method returns
`["ANTHROPIC_API_KEY"]` at runtime. These are separate concerns — do not confuse
`HarnessConfig.RequiredEnv` (used by custom harness and Phase 7) with
`Harness.RequiredEnvVars()` (used by the bridge to populate template data for all
harnesses including claude-code).

### Anti-Patterns to Avoid

- **Calling harness methods outside of bridge functions:** Phase 6 should call
  `BuildDockerfileData(h, cfg)`, not iterate `h.InstallCommands()` directly. Keeps
  all harness-to-template logic in one place.
- **Using reflection for key validation:** Direct field checks in Validate() are
  explicit and produce accurate per-harness error messages. Reflection adds complexity
  with no benefit here.
- **Adding NodeVersion/PythonVersion to the Harness interface:** These are config
  defaults from `MergedConfig.Harness`, not harness intrinsics. The bridge function
  resolves defaults from config. Harnesses express capability (NeedsNode bool),
  not version.
- **Panic in stub harnesses:** All stubs must return errors from Validate(), never
  panic. The spec is explicit: "not panics."

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Sorted harness name list in errors | Custom sort | `sort.Strings(names)` (stdlib) | Already in stdlib, one line |
| Config struct field enumeration | Reflection | Explicit field-by-field checks | Simpler, more accurate error messages |
| Version string concatenation | fmt.Sprintf | String concatenation | `"@anthropic-ai/claude-code" + "@" + version` is clearer |

**Key insight:** This phase is pure domain logic. The complexity is in getting all
the method return values right, not in infrastructure. Don't introduce abstractions
that aren't in the spec.

## Common Pitfalls

### Pitfall 1: HarnessConfig.RequiredEnv vs Harness.RequiredEnvVars()
**What goes wrong:** Confusing the custom harness's `RequiredEnv []string` config field
(which is "env vars the user says this custom tool requires, checked by Phase 7") with
the `RequiredEnvVars() []string` interface method (which is "env vars this harness needs
hardcoded, e.g. ANTHROPIC_API_KEY for claude-code").
**Why it happens:** Both deal with required env vars.
**How to avoid:** `HarnessConfig.RequiredEnv` feeds `Custom.RequiredEnvVars()` method
return value. `ClaudeCode.RequiredEnvVars()` returns `["ANTHROPIC_API_KEY"]` hardcoded.
**Warning signs:** Claude-code returning `cfg.RequiredEnv` instead of hardcoded slice.

### Pitfall 2: NodeVersion default resolution location
**What goes wrong:** Putting NodeVersion default logic ("if empty use 22") inside
`ClaudeCode` struct methods rather than in the bridge function.
**Why it happens:** Seems natural to keep it with the harness.
**How to avoid:** NodeVersion comes from `MergedConfig.Harness.NodeVersion` (set by
Phase 2 config merging). Default is resolved in `BuildDockerfileData` bridge function,
not in the harness struct. Harness only says `NeedsNode() = true`.
**Warning signs:** ClaudeCode struct with a `NodeVersion()` method.

### Pitfall 3: SkipPermissions extra_args injection location
**What goes wrong:** Adding `--dangerously-skip-permissions` to extra_args inside
`EntrypointCommand()` instead of via the bridge or at the point extra_args are appended.
**Why it happens:** Looks like it belongs with the command.
**How to avoid:** Per CONTEXT.md: "when skip_permissions = true, add
`--dangerously-skip-permissions` to extra_args." The bridge function
`BuildEntrypointData` reads `cfg.Harness.ExtraArgs` plus harness-specific appends.
The entrypoint template uses `exec {{ .EntrypointCommand }} "$@"` — the extra args
come in via the `$@` mechanism at container runtime, not baked into the command.
The bridge should populate `EntrypointData.EntrypointCommand` as the bare `claude`
command, and Phase 6 passes extra_args + skip_permissions flag to the Docker run
invocation.
**Warning signs:** `EntrypointCommand()` returning `"claude --dangerously-skip-permissions"`.

### Pitfall 4: import cycle between internal/harness and internal/docker
**What goes wrong:** Placing bridge functions in `internal/harness/` instead of
`internal/docker/`.
**Why it happens:** "Bridge between harness and templates" might seem like it belongs
with harnesses.
**How to avoid:** Import graph rule: `internal/docker -> internal/harness OK`. The
reverse direction is FORBIDDEN. All bridge functions live in `internal/docker/`.
**Warning signs:** `internal/harness/harness_bridge.go` importing `internal/docker`.

### Pitfall 5: Test file stub body
**What goes wrong:** `tests/harness_validate_test.go` is already a stub with only
`package tests`. Phase 5 must fill it in — not create a new file.
**Why it happens:** The file exists but has no test functions yet.
**How to avoid:** Edit the existing file rather than creating a new one.
**Warning signs:** Two `*_harness*` test files in the tests/ directory.

### Pitfall 6: Custom harness ConfigDirs vs Phase 7 mounts
**What goes wrong:** Implementing config dir mounting logic in Phase 5.
**Why it happens:** CONTEXT.md mentions `config_dirs` for custom harness.
**How to avoid:** Per CONTEXT.md: "`config_dirs` passed through to
`EntrypointData.ConfigCopyCommands` as copy-on-start commands (Phase 7 mounts the
actual host dirs)." Phase 5 only populates `ConfigCopyCommands` in `BuildEntrypointData`.
The actual Docker volume mounts are Phase 7's responsibility.

## Code Examples

### ClaudeCode harness — complete implementation sketch

```go
// Source: zone-spec.md §10 + CONTEXT.md locked decisions
type ClaudeCode struct {
    BaseHarness
    config *config.HarnessConfig
}

func (c *ClaudeCode) Name() string { return "claude-code" }

func (c *ClaudeCode) InstallCommands() []string {
    pkg := "@anthropic-ai/claude-code"
    if c.config.Version != "" {
        pkg += "@" + c.config.Version
    }
    return []string{"npm install -g " + pkg}
}

func (c *ClaudeCode) HealthCheck() string           { return "claude --version" }
func (c *ClaudeCode) EntrypointCommand() string      { return "claude" }
func (c *ClaudeCode) PromptFlag() string             { return "-p" }
func (c *ClaudeCode) RequiredEnvVars() []string      { return []string{"ANTHROPIC_API_KEY"} }
func (c *ClaudeCode) HomeConfigDir() string          { return "~/.claude" }
func (c *ClaudeCode) NeedsNode() bool                { return true }
func (c *ClaudeCode) NeedsPython() bool              { return false }
func (c *ClaudeCode) DefaultAptPackages() []string   { return nil }
func (c *ClaudeCode) DefaultNpmPackages() []string   { return nil }
func (c *ClaudeCode) DefaultPipPackages() []string   { return nil }
```

### Stub harness — exact error format

```go
// Source: zone-spec.md §10 lines 940-942
func (o *OpenCode) Validate() error {
    return fmt.Errorf(
        "the %q harness is not yet fully implemented; use harness = \"custom\" "+
        "with install_commands and entrypoint_command to configure it manually",
        o.Name(),
    )
}
```

### Cross-harness key error — exact format

```go
// Source: zone-spec.md §5 lines 587-589
return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
    "aider", "skip_permissions", "claude-code")
```

### BuildDockerfileData — key fields

```go
// Source: CONTEXT.md locked decisions
func BuildDockerfileData(h harness.Harness, cfg *config.MergedConfig) DockerfileData {
    nodeVer := cfg.Harness.NodeVersion
    if nodeVer == "" {
        nodeVer = "22"
    }
    return DockerfileData{
        BaseImage:              cfg.Zone.BaseImage,
        AptPackages:            append(cfg.Packages.Apt, h.DefaultAptPackages()...),
        NeedsNode:              h.NeedsNode(),
        NodeVersion:            nodeVer,
        NeedsPython:            h.NeedsPython(),
        NpmPackages:            append(cfg.Packages.Npm, h.DefaultNpmPackages()...),
        PipPackages:            append(cfg.Packages.Pip, h.DefaultPipPackages()...),
        HarnessInstallCommands: h.InstallCommands(),
        HealthCheck:            h.HealthCheck(),
        Shell:                  cfg.Zone.Shell,
        MountPath:              cfg.Workspace.MountPath,
    }
}
```

### BuildEntrypointData

```go
func BuildEntrypointData(h harness.Harness, cfg *config.MergedConfig) EntrypointData {
    // ConfigCopyCommands: one "cp -r <configDir> <dest>" per config dir
    var copyCmds []string
    homeDir := h.HomeConfigDir()
    if homeDir != "" {
        copyCmds = append(copyCmds, copyOnStartCmd(homeDir))
    }
    for _, d := range h.ExtraConfigDirs() {
        copyCmds = append(copyCmds, copyOnStartCmd(d))
    }
    name, email, forward := detectGitIdentity() // call platform.DetectGitIdentity
    return EntrypointData{
        MountPath:          cfg.Workspace.MountPath,
        ForwardGitConfig:   forward,
        GitUserName:        name,
        GitUserEmail:       email,
        ConfigCopyCommands: copyCmds,
        Shell:              cfg.Zone.Shell,
        EntrypointCommand:  h.EntrypointCommand(),
    }
}
```

### BuildShellRCData

```go
func BuildShellRCData(h harness.Harness, cfg *config.MergedConfig) ShellRCData {
    return ShellRCData{
        HarnessName:    h.Name(),
        MountPath:      cfg.Workspace.MountPath,
        Aliases:        h.Aliases(),
        ShellRC:        h.ShellRC(),
        WelcomeMessage: h.WelcomeMessage(),
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| map[string]interface{} for harness config | Typed HarnessConfig struct | Phase 2 (already done) | HAR-08 pre-satisfied |
| Stub files (package declaration only) | Full interface implementation | This phase | Harness system becomes functional |

**Already done by prior phases:**
- `HarnessConfig` typed struct: complete (Phase 2)
- All stub `.go` files in `internal/harness/`: exist with package declarations only
- `DockerfileData`, `EntrypointData`, `ShellRCData` structs: complete (Phase 4)
- `tests/harness_validate_test.go`: exists but empty (stub `package tests` only)

## Open Questions

1. **ConfigCopyCommands exact shell command format**
   - What we know: `EntrypointData.ConfigCopyCommands` is `[]string` of shell commands
     emitted verbatim in entrypoint.sh (see template: `{{- range .ConfigCopyCommands }}\n{{ . }}\n{{- end }}`)
   - What's unclear: The exact `cp` command format for copy-on-start (src path
     interpretation, whether to mkdir -p first)
   - Recommendation: Use `mkdir -p <dest_parent> && cp -r <src> <dest>` pattern.
     Phase 7 will refine mount semantics; Phase 5 just needs something valid in the
     template. Tests should verify the string is non-empty when HomeConfigDir != "".

2. **SkipPermissions extra_args timing**
   - What we know: `--dangerously-skip-permissions` should be added "to extra_args"
   - What's unclear: Added by which layer? The bridge? The harness? Phase 6?
   - Recommendation: Add in `BuildEntrypointData` by checking
     `cfg.Harness.SkipPermissions` when harness name is "claude-code" (or via a
     dedicated harness method). The template passes `$@` through, so Phase 6 must
     pass ExtraArgs at Docker run time, not baked into EntrypointCommand.
     Alternatively, `ClaudeCode.PostInstallCommands()` or a dedicated
     `ExtraRuntimeArgs() []string` method could be added. Safest: handle in
     `BuildEntrypointData` by inspecting harness name + cfg.Harness.SkipPermissions.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `go test` (testify v1.10.0) |
| Config file | none — `go test ./...` discovers all packages |
| Quick run command | `go test ./tests/ -run TestHarness -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HAR-01 | Harness interface has all required methods | unit | `go test ./tests/ -run TestHarnessInterface -v` | ❌ Wave 0 |
| HAR-02 | BaseHarness provides correct defaults | unit | `go test ./tests/ -run TestBaseHarnessDefaults -v` | ❌ Wave 0 |
| HAR-03 | Get() returns correct harness for known name | unit | `go test ./tests/ -run TestHarnessRegistry -v` | ❌ Wave 0 |
| HAR-03 | Get() returns error for unknown harness name | unit | `go test ./tests/ -run TestHarnessRegistryUnknown -v` | ❌ Wave 0 |
| HAR-04 | ClaudeCode returns correct install command | unit | `go test ./tests/ -run TestClaudeCodeInstall -v` | ❌ Wave 0 |
| HAR-04 | ClaudeCode with version sets @version suffix | unit | `go test ./tests/ -run TestClaudeCodeInstallVersioned -v` | ❌ Wave 0 |
| HAR-04 | ClaudeCode.RequiredEnvVars() returns ANTHROPIC_API_KEY | unit | `go test ./tests/ -run TestClaudeCodeEnvVars -v` | ❌ Wave 0 |
| HAR-04 | ClaudeCode.NeedsNode() returns true | unit | `go test ./tests/ -run TestClaudeCodeNeedsNode -v` | ❌ Wave 0 |
| HAR-05 | Stub harnesses return "not yet implemented" error | unit | `go test ./tests/ -run TestStubHarnessValidate -v` | ❌ Wave 0 |
| HAR-05 | Stub error message matches exact spec format | unit | `go test ./tests/ -run TestStubHarnessErrorMessage -v` | ❌ Wave 0 |
| HAR-06 | Custom with no entrypoint_command fails Validate() | unit | `go test ./tests/ -run TestCustomHarnessRequiresEntrypoint -v` | ❌ Wave 0 |
| HAR-06 | Custom with entrypoint_command passes Validate() | unit | `go test ./tests/ -run TestCustomHarnessValidates -v` | ❌ Wave 0 |
| HAR-06 | Custom install_commands optional (empty list valid) | unit | `go test ./tests/ -run TestCustomHarnessNoInstall -v` | ❌ Wave 0 |
| HAR-07 | skip_permissions on aider harness → specific error | unit | `go test ./tests/ -run TestCrossHarnessKeyValidation -v` | ❌ Wave 0 |
| HAR-07 | python_version on claude-code → specific error | unit | `go test ./tests/ -run TestCrossHarnessKeyValidation -v` | ❌ Wave 0 |
| HAR-07 | install_commands on claude-code → specific error | unit | `go test ./tests/ -run TestCrossHarnessKeyValidation -v` | ❌ Wave 0 |
| HAR-08 | HarnessConfig is typed struct (compile-time) | unit | `go build ./...` | ✅ (already in internal/config/) |
| HAR-09 | skip_permissions=nil treated as false | unit | `go test ./tests/ -run TestSkipPermissionsDefault -v` | ❌ Wave 0 |
| HAR-10 | ClaudeCode.PromptFlag() returns "-p" | unit | `go test ./tests/ -run TestPromptFlag -v` | ❌ Wave 0 |
| HAR-10 | Stubs return empty PromptFlag() | unit | `go test ./tests/ -run TestStubPromptFlag -v` | ❌ Wave 0 |
| Bridge | BuildDockerfileData populates NeedsNode/NodeVersion | unit | `go test ./tests/ -run TestBuildDockerfileData -v` | ❌ Wave 0 |
| Bridge | BuildEntrypointData populates EntrypointCommand | unit | `go test ./tests/ -run TestBuildEntrypointData -v` | ❌ Wave 0 |
| Bridge | BuildShellRCData populates HarnessName/Aliases | unit | `go test ./tests/ -run TestBuildShellRCData -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./tests/ -run TestHarness -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `tests/harness_validate_test.go` — fill existing stub; covers HAR-05, HAR-06,
  HAR-07, HAR-09, HAR-10
- [ ] `tests/harness_registry_test.go` — HAR-01, HAR-02, HAR-03 (interface + registry)
- [ ] `tests/harness_claude_code_test.go` — HAR-04 (ClaudeCode full method coverage)
- [ ] `tests/harness_bridge_test.go` — BuildDockerfileData, BuildEntrypointData,
  BuildShellRCData integration

---

## Sources

### Primary (HIGH confidence)
- `zone-spec.md §10` (lines 823-944) — Harness interface, BaseHarness, registry, typed
  HarnessConfig, stub behavior — read directly from source
- `zone-spec.md §5` (lines 543-594) — Harness-specific config keys, validation error
  format — read directly from source
- `internal/config/harness_config.go` — Confirmed HarnessConfig matches spec exactly
- `internal/docker/dockerfile.go` — Confirmed DockerfileData struct fields
- `internal/docker/entrypoint.go` — Confirmed EntrypointData struct fields
- `internal/docker/shellrc.go` — Confirmed ShellRCData struct fields
- `internal/harness/*.go` — Confirmed all stub files exist (package declarations only)

### Secondary (MEDIUM confidence)
- `zone-spec.md §7` (lines 717-727) — Import graph rules (confirmed
  `internal/docker -> internal/harness OK`)
- `zone-spec.md §11` (lines 947-1148) — Template source confirms `ConfigCopyCommands`,
  `EntrypointCommand`, `Aliases` field names match struct definitions

### Tertiary (LOW confidence)
- None — all findings verified from source files directly

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, no new packages, confirmed from go.mod
- Architecture: HIGH — spec §10 provides exact Go code for interface, BaseHarness,
  registry; CONTEXT.md locks all behavioral decisions
- Pitfalls: HIGH — identified from spec + existing code patterns (import graph,
  ConfigCopyCommands template, *bool merge semantics from Phase 2 decisions)

**Research date:** 2026-03-29
**Valid until:** 2026-04-28 (30 days — spec is stable, no external dependency changes)
