# Phase 2: Config Foundation - Research

**Researched:** 2026-03-27
**Domain:** Go TOML config parsing, two-tier merge, validation, and display
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Config output format (`zone config`)**
- Annotated TOML output with inline comments showing source per field
- Sources labeled as: `repo: zone.toml`, `global`, `global (default)`
- For merged lists (packages, env), annotate which elements came from global vs repo
- Valid TOML structure that's familiar and copy-pasteable

**JSON output (`zone config --json`)**
- Include source metadata per field, not just merged values
- Structure: each field is `{ "value": ..., "source": "repo|global|default" }`
- Scripts and tooling can inspect what's overridden vs inherited

**Validation error reporting**
- Collect ALL errors in one pass, don't stop at first
- Group errors by category: unknown keys, dangerous mounts, type errors, warnings
- Format: "zone.toml has N issues:" followed by grouped bullet points
- End with "Run zone validate after fixing."

**No-config behavior**
- `zone config` with no `zone.toml` in current directory: error with hint
- Message: "No zone.toml found. Run `zone init` to create one, or `zone config --global` to view global defaults."
- `zone validate` with no `zone.toml`: same error pattern
- `zone config --global`: always works (shows global config or defaults)

**Mount validation display**
- Show full symlink resolution chain: `~/docker.sock -> /var/run/docker.sock -> /run/docker.sock`
- Include reason WHY the path is blocked (e.g., "Docker socket mount allows container escape")
- Show all dangerous mount violations at once (not first-hit), since errors are collected
- `--allow-dangerous-mount` is a blanket flag (allows all blocked mounts in that invocation)

**Levenshtein suggestions**
- Show best single match only per unknown key (closest within Levenshtein distance 3)
- If no match within threshold, just report "unknown key" without suggestion
- Suggestions are section-aware: suggest `[harness].skip_permissions` not just `skip_permissions`

### Claude's Discretion
- Internal config struct design (nested structs, field tags, etc.)
- Exact Levenshtein implementation (library vs hand-rolled)
- Config file discovery order and XDG compliance details
- How defaults are represented internally (zero values vs explicit defaults)
- Test strategy and fixtures

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CFG-01 | User can create a minimal zone.toml with just `version = 1` and `harness = "claude-code"` | BurntSushi/toml Decode + MetaData.Undecoded() pattern; top-level `harness` sugar in types.go |
| CFG-02 | User can set global defaults in `~/.config/zone/config.toml` (XDG compliant) | os.UserConfigDir() returns $XDG_CONFIG_HOME or ~/.config; global.go reads path at startup |
| CFG-03 | Per-repo config overrides global for scalar fields | merge.go field-by-field copy: if repo field non-zero, take repo; else take global |
| CFG-04 | List fields merge correctly: packages union, network allow/deny append, extra_args append | merge.go per-field union/append/replace logic per spec §4.4 |
| CFG-05 | Unknown config keys produce an error with edit-distance suggestions (Levenshtein) | MetaData.Undecoded() returns []Key; agnivade/levenshtein.ComputeDistance(); distance threshold 3 |
| CFG-06 | Dangerous mount paths are blocked with symlink resolution | filepath.EvalSymlinks() resolves chain; blocklist in validate.go; display full chain |
| CFG-07 | `zone config` shows merged result with source annotations (global vs repo) | AnnotatedConfig type tracks value+source per field; cmd/config.go renders TOML with comments |
| CFG-08 | `zone config --json` outputs machine-readable merged config | JSON struct with `{ "value": ..., "source": "repo|global|default" }`; encoding/json |
| CFG-09 | Config schema version field (`version = 1`) is validated on parse | config.go checks version field after decode; missing = assume 1; unknown version = error |
| CFG-19 | Extra mounts default to read-only, require explicit `:rw` for write | validate.go parses extra_mounts strings; appends `:ro` if no permission suffix present |
</phase_requirements>

---

## Summary

Phase 2 builds the config foundation that every other phase depends on. The work is purely in `internal/config/` (6 files, all currently stubs) plus wiring two Cobra commands (`cmd/config.go`, `cmd/validate.go`). No Docker SDK involvement — this phase is pure Go.

The technical approach is straightforward: `github.com/BurntSushi/toml` handles TOML parsing and returns `MetaData` whose `Undecoded()` method surfaces unknown keys. `github.com/agnivade/levenshtein` provides edit-distance for suggestions. `path/filepath.EvalSymlinks()` resolves symlink chains for mount validation. `os.UserConfigDir()` provides XDG-compliant global config path without an external dependency.

The two-tier merge, source-annotated output, and multi-error collection pattern are the algorithmic core. None of it requires novel library research — the spec in `zone-spec.md` §4 is fully prescriptive.

**Primary recommendation:** Implement in strict file-by-file order: types.go → harness_config.go → config.go → global.go → merge.go → validate.go, then wire cmd/config.go and cmd/validate.go. Write tests alongside each file using the stub test files already in `tests/`.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/BurntSushi/toml | v1.6.0 | TOML parsing with unknown-key detection via MetaData.Undecoded() | Specified in zone-spec.md; largest Go TOML library; MetaData.Undecoded() is the correct strict-decode primitive |
| github.com/agnivade/levenshtein | v1.2.1 | Edit-distance for config key suggestions | Specified in project stack research; single function API, zero dependencies |
| path/filepath (stdlib) | stdlib | EvalSymlinks() for symlink chain resolution in mount validation | No external dependency needed; resolves entire chain to canonical path |
| os (stdlib) | stdlib | UserConfigDir() for XDG-compliant global config path | os.UserConfigDir() returns $XDG_CONFIG_HOME/zone or ~/.config/zone; no library needed |
| encoding/json (stdlib) | stdlib | `zone config --json` output | Struct tags, no external dependency |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| strings (stdlib) | stdlib | Mount string parsing (`/host:/container:ro`), path manipulation | For splitting extra_mounts format |
| fmt/errors (stdlib) | stdlib | Error wrapping with `%w`, sentinel errors | All validation errors bubble up wrapped |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| BurntSushi/toml MetaData.Undecoded() | pelletier/go-toml v2 | go-toml v2 has `Strict` decoder option but less widespread; BurntSushi is spec-required |
| agnivade/levenshtein | Hand-rolled Wagner-Fischer | Hand-roll is ~20 lines but adds maintenance; library is 50 lines, zero deps |
| os.UserConfigDir() | adrg/xdg library | Library adds dependency; stdlib is sufficient for `$XDG_CONFIG_HOME/zone/config.toml` path |

**Installation:**
```bash
# From /workspace/zone directory
go get github.com/BurntSushi/toml@v1.6.0
go get github.com/agnivade/levenshtein@v1.2.1
```

**Version verification (confirmed against Go module proxy 2026-03-27):**
- `github.com/BurntSushi/toml`: latest is v1.6.0 (previous research showed v1.5.0; v1.6.0 adds TOML 1.1 by default)
- `github.com/agnivade/levenshtein`: latest is v1.2.1

---

## Architecture Patterns

### Recommended File Execution Order

```
internal/config/
├── types.go           # 1st: Config, GlobalConfig, MergedConfig, AnnotatedConfig structs
├── harness_config.go  # 2nd: HarnessConfig typed union struct (consumed by all phases)
├── config.go          # 3rd: LoadRepo() — TOML parse + MetaData.Undecoded() check
├── global.go          # 4th: LoadGlobal() — XDG path + same parse pattern
├── merge.go           # 5th: Merge(global, repo) → MergedConfig — field-by-field rules
└── validate.go        # 6th: Validate(merged) → []ValidationError — mount check, version check

cmd/
├── config.go          # 7th: wire --json, --global flags; call config.LoadMerged + Show
└── validate.go        # 8th: wire RunE; call config.Validate; print grouped errors; exit 2
```

### Pattern 1: Strict TOML Decode via MetaData.Undecoded()

**What:** BurntSushi/toml does NOT have a `DecodeStrict()` function. The correct pattern is `Decode()` + check `MetaData.Undecoded()`.

**When to use:** In both `LoadRepo()` and `LoadGlobal()` — every TOML file parse.

**Example:**
```go
// Source: pkg.go.dev/github.com/BurntSushi/toml@v1.6.0
func loadTOMLStrict(path string, v any) (toml.MetaData, error) {
    md, err := toml.DecodeFile(path, v)
    if err != nil {
        return md, fmt.Errorf("parse %s: %w", path, err)
    }
    if undecoded := md.Undecoded(); len(undecoded) > 0 {
        // Convert []toml.Key to []string for error reporting
        keys := make([]string, len(undecoded))
        for i, k := range undecoded {
            keys[i] = k.String() // "zone.base_image" style
        }
        return md, &UnknownKeysError{Keys: keys, File: path}
    }
    return md, nil
}
```

**Critical note:** `toml.Key` is `[]string`. `k.String()` returns the dotted-path representation (e.g., `"harness.skip_perms"`). Use this string for Levenshtein comparison against the known key list.

### Pattern 2: Levenshtein Suggestions (section-aware)

**What:** For each unknown key, find the closest known key within distance 3.

**When to use:** Inside `UnknownKeysError` formatting or a dedicated `suggestKey()` helper.

**Example:**
```go
// Source: pkg.go.dev/github.com/agnivade/levenshtein@v1.2.1
import "github.com/agnivade/levenshtein"

// allKnownKeys is the flat list of all valid dotted key paths
var allKnownKeys = []string{
    "version",
    "harness",
    "zone.harness",
    "zone.base_image",
    "zone.shell",
    "harness.version",
    "harness.skip_permissions",
    "harness.node_version",
    "harness.python_version",
    "harness.extra_args",
    "harness.install_commands",
    "harness.entrypoint_command",
    "harness.config_dirs",
    "harness.required_env",
    "harness.health_check",
    "harness.aliases",
    "harness.shell_rc",
    "auth.mount_home_config",
    "auth.forward_env",
    "auth.forward_ssh_agent",
    "auth.env_file",
    "workspace.mount_path",
    "workspace.extra_mounts",
    "workspace.ports",
    "workspace.persist_home",
    "packages.apt",
    "packages.pip",
    "packages.npm",
    "resources.memory",
    "resources.cpus",
    "resources.pids_limit",
    "network.mode",
    "network.allow",
    "network.deny",
    "hooks.pre_build",
    "hooks.post_stop",
}

func suggestKey(unknown string) (string, bool) {
    best := ""
    bestDist := 4 // threshold: only suggest if distance <= 3
    for _, known := range allKnownKeys {
        d := levenshtein.ComputeDistance(unknown, known)
        if d < bestDist {
            bestDist = d
            best = known
        }
    }
    return best, best != ""
}
```

**Section-aware formatting:** When reporting, wrap suggestion in section brackets: if suggestion contains `.`, prefix the section. E.g., `"harness.skip_permissions"` → display as `[harness].skip_permissions`.

### Pattern 3: Two-Tier Merge

**What:** Field-by-field merge producing `MergedConfig`. Scalars: repo wins if non-zero. Lists: per-field union/append/replace rules from spec §4.4.

**When to use:** `merge.go` Merge() function. Called after both files load successfully.

**Example:**
```go
// Scalar override: repo wins if set (non-zero)
func mergeString(global, repo string) string {
    if repo != "" {
        return repo
    }
    return global
}

func mergeBool(global, repo *bool) *bool {
    if repo != nil {
        return repo
    }
    return global
}

// List union (packages): deduplicated, order preserved (global first)
func mergeUnion(global, repo []string) []string {
    seen := make(map[string]bool)
    result := make([]string, 0, len(global)+len(repo))
    for _, v := range append(global, repo...) {
        if !seen[v] {
            seen[v] = true
            result = append(result, v)
        }
    }
    return result
}

// List append (network.allow, hooks, extra_args): global + repo, no dedup
func mergeAppend(global, repo []string) []string {
    return append(global, repo...)
}

// Replace (extra_mounts, ports): repo replaces global entirely
func mergeReplace(global, repo []string) []string {
    if len(repo) > 0 {
        return repo
    }
    return global
}
```

### Pattern 4: Source Annotation for Display

**What:** Track where each value came from for `zone config` output.

**When to use:** Build alongside merge. AnnotatedField struct wraps value + source string.

**Example:**
```go
type Source string

const (
    SourceDefault Source = "global (default)"
    SourceGlobal  Source = "global"
    SourceRepo    Source = "repo: zone.toml"
)

type AnnotatedField[T any] struct {
    Value  T
    Source Source
}

// For JSON output (zone config --json)
// Each field serializes as {"value": ..., "source": "repo|global|default"}
type AnnotatedFieldJSON struct {
    Value  any    `json:"value"`
    Source Source `json:"source"`
}
```

**For annotated list fields** (packages, forward_env): track which elements came from where, then render inline comments per element group.

### Pattern 5: Multi-Error Collection

**What:** Run all validation checks, collect errors into a slice, report all at once.

**When to use:** In `validate.go` Validate() function and at the end of merge.

**Example:**
```go
type ValidationError struct {
    Category string // "unknown_key", "dangerous_mount", "type_error", "warning"
    Message  string
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
    if len(ve) == 0 {
        return ""
    }
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("zone.toml has %d issues:\n", len(ve)))
    // group by category, then bullet points
    // End with "Run zone validate after fixing."
    return sb.String()
}
```

### Pattern 6: Mount Validation with Symlink Chain

**What:** Resolve each extra_mount host path through the full symlink chain, then check against blocklist. Display the full chain in the error message.

**When to use:** In `validate.go` for each element of `extra_mounts`.

**Example:**
```go
// Source: pkg.go.dev/path/filepath#EvalSymlinks
func validateMount(hostPath string) error {
    resolved, err := filepath.EvalSymlinks(hostPath)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("cannot resolve mount path %s: %w", hostPath, err)
    }
    // Build display chain for error message
    chain := buildSymlinkChain(hostPath) // walk os.Readlink step by step

    if isBlockedMount(resolved) {
        reason := blockedMountReason(resolved)
        return &DangerousMountError{
            Path:    hostPath,
            Resolved: resolved,
            Chain:   chain,
            Reason:  reason,
        }
    }

    // Warning: mount outside current project directory
    cwd, _ := os.Getwd()
    if !strings.HasPrefix(resolved, cwd) {
        // append to warnings, not errors
    }
    return nil
}

var dangerousMountBlocklist = []mountRule{
    {pattern: "/var/run/docker.sock", reason: "Docker socket mount allows container escape"},
    {pattern: "/var/run/podman/",     reason: "Container runtime socket mount"},
    {pattern: "/var/run/containerd/", reason: "Container runtime socket mount"},
    {pattern: "/proc",                reason: "Kernel interface mount"},
    {pattern: "/sys",                 reason: "Kernel interface mount"},
    {pattern: "/dev",                 reason: "Device mount"},
    {pattern: "/.ssh",                reason: "SSH keys exposure (use forward_ssh_agent instead)"},
    {pattern: "/etc/shadow",          reason: "Host credentials file"},
    {pattern: "/etc/passwd",          reason: "Host credentials file"},
    {pattern: "/",                    reason: "Host root mount"},
    {pattern: "/etc",                 reason: "Host system config mount"},
    {pattern: "/.kube",               reason: "Kubernetes credentials exposure"},
    {pattern: "/.aws",                reason: "AWS credentials exposure"},
    {pattern: "/.gcp",                reason: "GCP credentials exposure"},
    {pattern: "/.azure",              reason: "Azure credentials exposure"},
    {pattern: "/.docker",             reason: "Docker credentials exposure"},
    {pattern: "/.gnupg",              reason: "GPG key exposure"},
    {pattern: "/boot",                reason: "Kernel boot files"},
    {pattern: "/lib/modules",         reason: "Kernel modules"},
}
```

### Pattern 7: Extra Mounts Default to Read-Only (CFG-19)

**What:** Parse mount strings; if no permission suffix (`:ro` or `:rw`), append `:ro` before passing to Docker.

**When to use:** In `validate.go` or a mount normalization helper called during merge.

**Example:**
```go
func normalizeMountPermission(mountSpec string) string {
    parts := strings.Split(mountSpec, ":")
    switch len(parts) {
    case 2: // "/host:/container" — no permission
        return mountSpec + ":ro"
    case 3: // "/host:/container:ro|rw"
        perm := parts[2]
        if perm != "ro" && perm != "rw" {
            // invalid permission — validation error
        }
        return mountSpec
    default:
        // invalid format — validation error
        return mountSpec
    }
}
```

### Pattern 8: XDG Global Config Path

**What:** Use `os.UserConfigDir()` which returns `$XDG_CONFIG_HOME` or `~/.config` — no external library needed.

**When to use:** In `global.go` GlobalConfigPath().

**Example:**
```go
func GlobalConfigPath() (string, error) {
    base, err := os.UserConfigDir()
    if err != nil {
        return "", fmt.Errorf("global config: %w", err)
    }
    return filepath.Join(base, "zone", "config.toml"), nil
}
```

### Anti-Patterns to Avoid

- **Calling toml.Decode without checking Undecoded():** Silent unknown key acceptance; the pitfall from PITFALLS.md §11 explicitly calls this out.
- **Stopping at first validation error:** Phase requires all errors collected in one pass. Use `ValidationErrors` slice pattern.
- **Hand-rolling XDG path detection:** `os.UserConfigDir()` handles `$XDG_CONFIG_HOME` correctly on all platforms.
- **Using `map[string]interface{}` for HarnessConfig:** Spec §10 explicitly requires typed struct. Per-harness `Validate()` can only work with typed fields.
- **Checking mount paths without EvalSymlinks:** Symlink escape is the exact security vulnerability; always resolve before checking.
- **Blocking on missing global config file:** If `~/.config/zone/config.toml` does not exist, use zero-value GlobalConfig (all defaults). Only error if file exists but fails to parse.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TOML parsing | Custom parser | github.com/BurntSushi/toml | TOML spec compliance, MetaData API for unknown keys |
| Edit distance | Custom Levenshtein | github.com/agnivade/levenshtein | Correct Unicode handling (operates on runes), tested |
| XDG path | Env var inspection | os.UserConfigDir() | stdlib, correct on Linux + macOS + Windows |
| Symlink resolution | os.Readlink loop | filepath.EvalSymlinks() | Resolves entire chain to canonical path atomically |

**Key insight:** The hard part of this phase is the merge algorithm and the annotated output format — both are specification work, not library work. The libraries are thin utilities.

---

## Common Pitfalls

### Pitfall 1: MetaData.Undecoded() Misses Keys Behind Nested Structs

**What goes wrong:** `Undecoded()` documentation states: "keys hidden behind Primitive values" are not included until decoded. If a field uses `toml.Primitive`, unknown sub-keys are not caught.

**Why it happens:** Developers use `Primitive` for deferred decoding (e.g., harness section). Unknown keys in the deferred section are invisible to the initial `Undecoded()` call.

**How to avoid:** Do NOT use `toml.Primitive` for any config section. Decode the full struct in one pass using typed structs. `HarnessConfig` must be a typed struct (which it is per spec) so all its keys are decoded and unknown ones surface in `Undecoded()`.

**Warning signs:** Any `toml.Primitive` field in Config or HarnessConfig structs.

### Pitfall 2: toml.Key.String() Returns Lowercase Dotted Path

**What goes wrong:** `toml.Key.String()` returns the key as it appears in the TOML file. TOML keys are case-sensitive, and the String() result for a nested key like `[harness] skip_perms` is `"harness.skip_perms"` — the dotted path. The known keys list for Levenshtein must use the same dotted format.

**Why it happens:** Developers build the known-key list with section-aware names like `"[harness].skip_permissions"` (with brackets) while Undecoded() returns `"harness.skip_permissions"` (without brackets). The distance is then inflated by the bracket characters, causing missed suggestions.

**How to avoid:** Build `allKnownKeys` without brackets: `"harness.skip_permissions"` not `"[harness].skip_permissions"`. Format with brackets only in the error message display layer.

### Pitfall 3: Missing Global Config File Should Not Error

**What goes wrong:** `global.go` returns an error when `~/.config/zone/config.toml` doesn't exist. First-time users and CI environments never create this file. The error cascades through `LoadMerged()` and prevents any zone command from running.

**Why it happens:** Standard file-open error handling treats "not found" the same as "parse error."

**How to avoid:**
```go
func LoadGlobal() (*GlobalConfig, error) {
    path, err := GlobalConfigPath()
    if err != nil {
        return defaultGlobalConfig(), nil
    }
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return defaultGlobalConfig(), nil  // no file = use defaults, not error
    }
    // ... parse file
}
```

### Pitfall 4: Annotated Output Breaks on Large List Fields

**What goes wrong:** The annotated TOML output for `packages.apt` with 20 elements and mixed sources requires generating per-element inline comments. Naive string building produces malformed TOML (inline comments on list elements are not valid TOML).

**Why it happens:** TOML does not support inline comments on individual array elements. The spec example uses a line comment above the array showing which part is global vs repo.

**How to avoid:** For merged list fields, emit a comment block above the array, then the array on the next lines. Example:
```toml
[packages]
# apt: global provides ["git", "curl", "wget"]; repo adds ["build-essential"]
apt = ["git", "curl", "wget", "build-essential"]
```
Not:
```toml
apt = [
  "git",    # global -- INVALID TOML inline comment
]
```

### Pitfall 5: EvalSymlinks Fails on Non-Existent Paths

**What goes wrong:** `filepath.EvalSymlinks(path)` returns an error if the path doesn't exist. Mount paths in config may reference paths not yet created (e.g., `/data/models` on a build machine). Erroring here blocks `zone validate` unnecessarily.

**Why it happens:** The function requires the path to exist to resolve symlinks. Path existence is not a validation concern for future mounts.

**How to avoid:** Check `os.IsNotExist(err)` after `EvalSymlinks`. If the path doesn't exist, use the original path for blocklist checking (can't have a dangerous symlink if path doesn't exist). Only error if the resolution fails for a reason other than non-existence.

### Pitfall 6: Version Field as Wrong TOML Type

**What goes wrong:** Users write `version = "1"` (string) instead of `version = 1` (integer). BurntSushi/toml rejects this with a type error — but the error message from the library is cryptic ("cannot unmarshal TOML string into Go struct field...").

**Why it happens:** YAML and JSON configs typically use strings for version. TOML is type-strict.

**How to avoid:** In the config validation pass, if version parse fails, check if the raw TOML contains `version = "` (string form) and emit a targeted error: "version must be an integer (use `version = 1`, not `version = \"1\"`).

### Pitfall 7: Levenshtein on Fully Qualified vs Short Names

**What goes wrong:** User writes `skip_permissions = true` at top level (not under `[harness]`). The unknown key is `"skip_permissions"` (no prefix). The best Levenshtein match against `"harness.skip_permissions"` has distance 8 (prefix adds too much), so no suggestion fires.

**Why it happens:** The unknown key is a bare name but the known key is section-qualified. The distance is dominated by the section prefix.

**How to avoid:** Run Levenshtein against both the fully-qualified known keys AND the bare key names (last component of each path). If bare-name match fires, display with the full section path as the suggestion: "Did you mean `[harness] skip_permissions`?"

---

## Code Examples

Verified patterns from official sources and project spec:

### Loading and Strict-Decoding a TOML File
```go
// Source: pkg.go.dev/github.com/BurntSushi/toml@v1.6.0
md, err := toml.DecodeFile(path, &cfg)
if err != nil {
    return nil, fmt.Errorf("parse %s: %w", path, err)
}
// Unknown key detection
if keys := md.Undecoded(); len(keys) > 0 {
    return nil, newUnknownKeysError(path, keys)
}
```

### Computing Edit Distance
```go
// Source: pkg.go.dev/github.com/agnivade/levenshtein@v1.2.1
import "github.com/agnivade/levenshtein"

dist := levenshtein.ComputeDistance("skip_perms", "skip_permissions")
// dist == 5
dist2 := levenshtein.ComputeDistance("baes_image", "base_image")
// dist2 == 1 — within threshold, suggest "base_image"
```

### XDG Config Path
```go
// Source: pkg.go.dev/os#UserConfigDir
base, err := os.UserConfigDir()
// Linux: $XDG_CONFIG_HOME or ~/.config
// macOS: ~/Library/Application Support (NOTE: not ~/.config)
// Zone uses Linux/macOS, so fine for ~/.config/zone/config.toml on Linux
// On macOS the path will be ~/Library/Application Support/zone/config.toml
path := filepath.Join(base, "zone", "config.toml")
```

**macOS note:** `os.UserConfigDir()` on macOS returns `~/Library/Application Support`, NOT `~/.config`. The spec says "XDG compliant" with `~/.config` as the example. For macOS compatibility while honoring the spec intent, check `$XDG_CONFIG_HOME` first:
```go
func GlobalConfigPath() (string, error) {
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "zone", "config.toml"), nil
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".config", "zone", "config.toml"), nil
}
```
This matches the spec literal (`~/.config/zone/`) on both Linux and macOS, regardless of macOS's `UserConfigDir()` behavior.

### Resolving Symlinks for Mount Validation
```go
// Source: pkg.go.dev/path/filepath#EvalSymlinks
resolved, err := filepath.EvalSymlinks(hostPath)
if err != nil {
    if os.IsNotExist(err) {
        resolved = hostPath // path doesn't exist yet; check as-is
    } else {
        return fmt.Errorf("cannot resolve %s: %w", hostPath, err)
    }
}
```

### Typed HarnessConfig Struct (from zone-spec.md §10)
```go
// Source: zone-spec.md §10 Typed HarnessConfig
type HarnessConfig struct {
    // Common
    Version   string   `toml:"version"`
    ExtraArgs []string `toml:"extra_args"`

    // Claude Code
    SkipPermissions *bool  `toml:"skip_permissions"`
    NodeVersion     string `toml:"node_version"`

    // Aider
    PythonVersion string `toml:"python_version"`

    // Custom
    InstallCommands   []string          `toml:"install_commands"`
    EntrypointCommand string            `toml:"entrypoint_command"`
    ConfigDirs        []string          `toml:"config_dirs"`
    RequiredEnv       []string          `toml:"required_env"`
    CustomHealthCheck string            `toml:"health_check"`
    CustomAliases     map[string]string `toml:"aliases"`
    CustomShellRC     []string          `toml:"shell_rc"`
}
```

Note: `SkipPermissions` uses `*bool` (pointer) so that `nil` = "not set by user" vs `false` = "user explicitly set to false". This enables merge logic to distinguish "override" from "absent."

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| BurntSushi/toml `DecodeStrict()` | Does not exist — use `Decode()` + `MetaData.Undecoded()` | Never existed | Developers must explicitly check Undecoded() |
| BurntSushi/toml v1.5.0 | v1.6.0 (TOML 1.1 enabled by default, float encoding fix) | Dec 2024 | Safe to upgrade; no breaking API changes |
| os.UserConfigDir() for XDG on macOS | XDG_CONFIG_HOME env var check first | Always | macOS returns ~/Library, not ~/.config |
| agnivade/levenshtein v1.2.0 | v1.2.1 (latest) | Minor patch | Use v1.2.1 |

**No deprecated items relevant to this phase.**

---

## Open Questions

1. **HarnessConfig validation scope in this phase**
   - What we know: Phase 2 requires type definitions only for harness config (per CONTEXT.md canonical refs §5 and §10)
   - What's unclear: Should `validate.go` call harness-specific `Validate()` methods (cross-field checks like "aider doesn't support skip_permissions"), or does that belong to Phase 5 (Harness System)?
   - Recommendation: Phase 2 validates config structure (unknown keys, types, mounts). Harness-specific cross-field validation (e.g., wrong harness for a field) belongs in Phase 5. Phase 2 validates that the `[harness]` section keys exist in `HarnessConfig` struct — that's all BurntSushi/toml's Undecoded() already provides.

2. **harness sugar field in top-level vs [zone] section**
   - What we know: Spec §4.2 says top-level `harness = "claude-code"` is sugar for `[zone] harness`. BurntSushi/toml will decode top-level `harness` only if the struct has a top-level `Harness` field.
   - What's unclear: How to handle both `harness` at top-level AND `[zone].harness` without duplicating fields or causing unknown-key errors.
   - Recommendation: Add a top-level `Harness string \`toml:"harness"\`` field in the repo `Config` struct. After parsing, if non-empty, copy to `Zone.Harness`. Run `Undecoded()` check after this normalization step.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard testing (`go test`) |
| Config file | None needed — uses Go test flags |
| Quick run command | `go test ./tests/ -run TestConfigMerge -v` |
| Full suite command | `go test ./tests/ -v -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CFG-01 | Minimal zone.toml (`version=1`, `harness="claude-code"`) parses without error | unit | `go test ./tests/ -run TestMinimalConfig -v` | ❌ Wave 0 |
| CFG-02 | Global config loaded from XDG path | unit | `go test ./tests/ -run TestGlobalConfigLoad -v` | ❌ Wave 0 |
| CFG-03 | Scalar fields: repo overrides global | unit | `go test ./tests/ -run TestScalarOverride -v` | ❌ Wave 0 |
| CFG-04 | List fields: union/append/replace merge | unit | `go test ./tests/ -run TestListMerge -v` | ❌ Wave 0 |
| CFG-05 | Unknown key produces Levenshtein suggestion | unit | `go test ./tests/ -run TestUnknownKeySuggestion -v` | ❌ Wave 0 |
| CFG-06 | Dangerous mount blocked with symlink chain in error | unit | `go test ./tests/ -run TestDangerousMount -v` | ❌ Wave 0 |
| CFG-07 | `zone config` output has source annotations | integration | `go test ./tests/ -run TestConfigAnnotatedOutput -v` | ❌ Wave 0 |
| CFG-08 | `zone config --json` has value+source structure | unit | `go test ./tests/ -run TestConfigJSON -v` | ❌ Wave 0 |
| CFG-09 | version=1 valid; unsupported version errors; missing version defaults to 1 | unit | `go test ./tests/ -run TestConfigVersion -v` | ❌ Wave 0 |
| CFG-19 | extra_mounts without permission suffix get :ro appended | unit | `go test ./tests/ -run TestMountReadOnly -v` | ❌ Wave 0 |

All tests go in existing stub files:
- `tests/config_merge_test.go` — CFG-01, CFG-02, CFG-03, CFG-04, CFG-07, CFG-08, CFG-09
- `tests/validate_test.go` — CFG-05, CFG-06, CFG-19

### Sampling Rate
- **Per task commit:** `go test ./tests/ -run TestConfig -v`
- **Per wave merge:** `go test ./tests/ -v -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `tests/config_merge_test.go` — implement test functions (file exists as stub: `package tests`)
- [ ] `tests/validate_test.go` — implement test functions (file exists as stub: `package tests`)
- [ ] `go get github.com/BurntSushi/toml@v1.6.0` — not yet in go.mod
- [ ] `go get github.com/agnivade/levenshtein@v1.2.1` — not yet in go.mod

---

## Sources

### Primary (HIGH confidence)
- `zone-spec.md §4` (local) — Complete TOML schemas, merge strategy, validation rules, dangerous mount list, error message examples
- `zone-spec.md §5` (local) — HarnessConfig typed struct definition
- `zone-spec.md §7` (local) — Project structure, file responsibilities
- `zone-spec.md §8` (local) — Error handling convention (sentinel errors, wrapping)
- `pkg.go.dev/github.com/BurntSushi/toml@v1.6.0` — Decode/DecodeFile signatures, MetaData.Undecoded() API, Key type
- `pkg.go.dev/github.com/agnivade/levenshtein@v1.2.1` — ComputeDistance signature
- `pkg.go.dev/path/filepath#EvalSymlinks` — EvalSymlinks signature and behavior
- `pkg.go.dev/os#UserConfigDir` — XDG config path behavior per platform

### Secondary (MEDIUM confidence)
- `.planning/research/STACK.md` (local) — Stack decisions; BurntSushi/toml specified, levenshtein library identified
- `.planning/research/PITFALLS.md §11` (local) — TOML strict decode pitfall, merge validation sequence
- Go module proxy (`go list -m -versions`) — Confirmed latest BurntSushi/toml=v1.6.0, levenshtein=v1.2.1

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against module proxy; APIs verified against pkg.go.dev
- Architecture: HIGH — patterns derived from spec §4/5/8 (authoritative) plus verified library APIs
- Pitfalls: HIGH — pitfall §11 from prior research explicitly covers this phase; others derived from API documentation gaps

**Research date:** 2026-03-27
**Valid until:** 2026-04-27 (stable libraries; BurntSushi/toml and levenshtein are not fast-moving)
