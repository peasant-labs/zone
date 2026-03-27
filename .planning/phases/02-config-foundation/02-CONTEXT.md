# Phase 2: Config Foundation - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Two-tier TOML config system: parse `zone.toml` (per-repo) and `~/.config/zone/config.toml` (global), merge them with field-specific rules, validate with strict decoding and dangerous mount detection, and display merged results with source annotations. Requirements: CFG-01 through CFG-09, CFG-19.

</domain>

<decisions>
## Implementation Decisions

### Config output format (`zone config`)
- Annotated TOML output with inline comments showing source per field
- Sources labeled as: `repo: zone.toml`, `global`, `global (default)`
- For merged lists (packages, env), annotate which elements came from global vs repo
- Valid TOML structure that's familiar and copy-pasteable

### JSON output (`zone config --json`)
- Include source metadata per field, not just merged values
- Structure: each field is `{ "value": ..., "source": "repo|global|default" }`
- Scripts and tooling can inspect what's overridden vs inherited

### Validation error reporting
- Collect ALL errors in one pass, don't stop at first
- Group errors by category: unknown keys, dangerous mounts, type errors, warnings
- Format: "zone.toml has N issues:" followed by grouped bullet points
- End with "Run zone validate after fixing."

### No-config behavior
- `zone config` with no `zone.toml` in current directory: error with hint
- Message: "No zone.toml found. Run `zone init` to create one, or `zone config --global` to view global defaults."
- `zone validate` with no `zone.toml`: same error pattern
- `zone config --global`: always works (shows global config or defaults)

### Mount validation display
- Show full symlink resolution chain: `~/docker.sock -> /var/run/docker.sock -> /run/docker.sock`
- Include reason WHY the path is blocked (e.g., "Docker socket mount allows container escape")
- Show all dangerous mount violations at once (not first-hit), since errors are collected
- `--allow-dangerous-mount` is a blanket flag (allows all blocked mounts in that invocation)

### Levenshtein suggestions
- Show best single match only per unknown key (closest within Levenshtein distance 3)
- If no match within threshold, just report "unknown key" without suggestion
- Suggestions are section-aware: suggest `[harness].skip_permissions` not just `skip_permissions`

### Claude's Discretion
- Internal config struct design (nested structs, field tags, etc.)
- Exact Levenshtein implementation (library vs hand-rolled)
- Config file discovery order and XDG compliance details
- How defaults are represented internally (zero values vs explicit defaults)
- Test strategy and fixtures

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Config system (primary)
- `zone-spec.md` section 4 — Full TOML schemas for both config tiers, merge strategy per field, validation rules, edit-distance suggestions, dangerous mount list, environment forwarding, SSH agent forwarding
- `zone-spec.md` section 4.1 — Global config schema (complete TOML example with all fields)
- `zone-spec.md` section 4.2 — Per-repo config schema (minimal and full examples, `harness` sugar)
- `zone-spec.md` section 4.3 — Schema versioning rules (version field handling)
- `zone-spec.md` section 4.4 — Merge strategy (scalar override, list union/append/replace per field)
- `zone-spec.md` section 4.5 — Validation rules (BurntSushi/toml strict decode, mount validation, additional validations)

### Harness config (Phase 2 scope: type definitions only)
- `zone-spec.md` section 5 — Harness-specific config keys by harness type (typed struct, not map)
- `zone-spec.md` section 10 `Typed HarnessConfig` — Go struct definition for harness config

### Error handling
- `zone-spec.md` section 8 — Error handling convention (sentinel errors, wrapping, exit codes)

### Project structure
- `zone-spec.md` section 7 — Import graph rules, package layout (config package dependencies)

### Project context
- `.planning/PROJECT.md` — Tech stack constraints (BurntSushi/toml specified), key decisions
- `.planning/REQUIREMENTS.md` — CFG-01 through CFG-09, CFG-19 requirement details

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/config.go`: Cobra command stub — needs `RunE` implementation to call config loading + display
- `cmd/validate.go`: Cobra command stub — needs `RunE` implementation to call validation
- `cmd/root.go`: Root command with all subcommands registered
- `internal/config/`: 6 stub files (types.go, config.go, merge.go, validate.go, global.go, harness_config.go) — package declared, doc comments only

### Established Patterns
- Cobra command pattern: `RunE` returning `fmt.Errorf("not implemented")` — replace with real implementations
- Package layout from spec §7: `internal/config/` is the home for all config code
- Go 1.24, module path `github.com/peasant-labs/zone`

### Integration Points
- `cmd/config.go` and `cmd/validate.go` will call into `internal/config/` package
- `internal/config/types.go` defines structs consumed by ALL downstream phases (harness, docker, cache, network)
- Config types are the foundation import — every other internal package will depend on them
- `tests/config_merge_test.go` and `tests/validate_test.go` stub files exist for test placement

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the spec — the spec is very prescriptive for this phase with complete TOML schemas, merge rules, and error message examples.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-config-foundation*
*Context gathered: 2026-03-27*
