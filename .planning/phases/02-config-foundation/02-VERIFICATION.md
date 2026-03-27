---
phase: 02-config-foundation
verified: 2026-03-27T08:00:00Z
status: passed
score: 5/5 must-haves verified
---

# Phase 02: Config Foundation Verification Report

**Phase Goal:** Users can write zone.toml and ~/.config/zone/config.toml and have them merged, validated, and surfaced clearly on errors
**Verified:** 2026-03-27
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A minimal `zone.toml` with `version = 1` and `harness = "claude-code"` parses without error | VERIFIED | `TestMinimalConfig` passes; `LoadRepo` two-phase decode handles harness sugar; `TestConfigVersion` confirms version=1 valid |
| 2 | Global config and per-repo config merge correctly: scalars overridden by repo, lists unioned | VERIFIED | `TestScalarOverride`, `TestScalarFallback`, `TestListUnion`, `TestNetworkAllow`, `TestSourceAnnotation` all pass; `Merge()` in merge.go implements all spec 4.4 rules |
| 3 | An unknown config key produces an error message with a Levenshtein edit-distance suggestion | VERIFIED | `TestUnknownKeySuggestion_Close`, `TestUnknownKeySuggestion_BareKey`, `TestUnknownKeySuggestion_SectionAware` pass; `SuggestKey()` three-pass strategy in validate.go |
| 4 | A dangerous mount path (e.g., docker.sock) is blocked with a clear error including the resolved symlink | VERIFIED | `TestDangerousMount_DockerSocket`, `TestDangerousMount_SymlinkResolution`, `TestDangerousMount_AllCollected` pass; `Validate()` in validate.go with `buildSymlinkChain()` |
| 5 | `zone config` prints the merged config with each value annotated as global or repo source | VERIFIED | `TestConfigAnnotatedOutput` and `TestConfigAnnotatedOutput_ListMerge` pass; `renderAnnotatedTOML()` in cmd/config.go produces `# <source>` comments |

**Score: 5/5 truths verified**

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/config/types.go` | Config structs, Source constants, AnnotatedField generic, DefaultRepoConfig/DefaultGlobalConfig | VERIFIED | All exports present: `Source`, `SourceDefault`, `SourceGlobal`, `SourceRepo`, `AnnotatedField[T]`, `RepoConfig`, `GlobalConfig`, `MergedConfig`, `AnnotatedConfig`, `HarnessName string`, `DefaultRepoConfig()`, `DefaultGlobalConfig()` |
| `internal/config/harness_config.go` | HarnessConfig typed struct with per-harness fields | VERIFIED | `HarnessConfig` with `SkipPermissions *bool`, `InstallCommands []string`, `EntrypointCommand string` — no `map[string]interface{}` |
| `internal/config/config.go` | `LoadRepo()` TOML parser with strict decode | VERIFIED | `LoadRepo`, `UnknownKeysError`, `ErrNoConfig`, `ErrVersionMismatch`, `md.Undecoded()`, `repoConfigSugar`, version validation with `cfg.Version == 0` defaulting to 1 |
| `internal/config/global.go` | `LoadGlobal()` with XDG path and missing-file handling | VERIFIED | `GlobalConfigPath()` uses `$XDG_CONFIG_HOME` first, then `~/.config`; `LoadGlobal()` returns defaults on missing file via `os.IsNotExist` |
| `internal/config/merge.go` | `Merge()` producing MergedConfig + AnnotatedConfig with source tracking; `LoadMerged()` | VERIFIED | Full spec 4.4 semantics: `mergeString`, `mergeBoolPtr`, `mergeIntAnnotated`, `mergeUnion`, `mergeAppend`, `mergeReplace`, `mergeStringMaps`; `LoadMerged()` convenience function |
| `internal/config/validate.go` | `Validate()` with mount checking, Levenshtein suggestions, multi-error collection | VERIFIED | `Validate()`, `SuggestKey()`, `FormatSuggestion()`, `NormalizeMountPermission()`, `ValidateUnknownKeys()`, `ValidationErrors`, `DangerousMountError`, `HasErrors()`, `Warnings()` |
| `cmd/config.go` | `zone config` command with --json and --global flags | VERIFIED | `configCmd` with `--json` and `--global` flags; `renderAnnotatedTOML()`, `renderJSON()`, `config.LoadMerged`, `config.LoadGlobal`, `config.ErrNoConfig`, error message "No zone.toml found. Run `zone init`..." |
| `cmd/validate.go` | `zone validate` command with exit code 2 on errors | VERIFIED | `validateCmd` with `--allow-dangerous-mount`; `config.LoadRepo`, `config.Validate`, `config.ValidateUnknownKeys`, `errors.As` for UnknownKeysError, `os.Exit(2)`, "zone.toml is valid" success message |
| `tests/config_merge_test.go` | Tests for CFG-01, CFG-03, CFG-04, CFG-09 merge behaviors | VERIFIED | 13 tests: `TestMinimalConfig`, `TestScalarOverride`, `TestScalarFallback`, `TestListUnion`, `TestListAppend`, `TestNetworkAllow`, `TestListReplace`, `TestHooksAppend`, `TestExtraArgsAppend`, `TestBoolOverride`, `TestBoolNilFallback`, `TestConfigVersion`, `TestSourceAnnotation` — all pass |
| `tests/validate_test.go` | Tests for CFG-05, CFG-06, CFG-19 validation behaviors | VERIFIED | 16 tests: `TestUnknownKeySuggestion_*`, `TestDangerousMount_*`, `TestMountReadOnly_*`, `TestBaseImageWarning`, `TestNetworkModeNoneWithAllow`, `TestMultipleErrors` — all pass |
| `tests/config_cmd_test.go` | Integration tests for annotated TOML and JSON output (CFG-07, CFG-08) | VERIFIED | 6 tests: `TestConfigAnnotatedOutput`, `TestConfigAnnotatedOutput_ListMerge`, `TestConfigAnnotatedOutput_GlobalOnly`, `TestConfigAnnotatedOutput_NoZoneToml`, `TestConfigJSON`, `TestConfigJSON_Structure` — all pass |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/config/config.go` | `internal/config/types.go` | imports `RepoConfig` struct for TOML decode target | WIRED | `RepoConfig` used as decode target in `LoadRepo()`; `repoConfigSugar` internal struct mirrors it |
| `internal/config/global.go` | `internal/config/types.go` | imports `GlobalConfig` struct for TOML decode target | WIRED | `GlobalConfig` used as decode target in `LoadGlobal()` |
| `internal/config/merge.go` | `internal/config/types.go` | uses `RepoConfig`, `GlobalConfig`, `MergedConfig`, `AnnotatedConfig`, `Source` constants | WIRED | `MergedConfig` and `AnnotatedConfig` populated in `Merge()`; all `SourceDefault`, `SourceGlobal`, `SourceRepo` used |
| `internal/config/validate.go` | `github.com/agnivade/levenshtein` | calls `levenshtein.ComputeDistance()` in `SuggestKey()` | WIRED | Direct import on line 11; `ComputeDistance` called in three-pass strategy; `go mod tidy` confirms as direct dependency |
| `cmd/config.go` | `internal/config` | calls `LoadMerged()`, `LoadGlobal()`, renders `AnnotatedConfig` | WIRED | `config.LoadMerged`, `config.LoadGlobal`, `config.Merge` called; `renderAnnotatedTOML(annotated)` and `renderJSON(annotated)` consume `*config.AnnotatedConfig` |
| `cmd/validate.go` | `internal/config` | calls `LoadRepo()`, `Validate()`, `ValidateUnknownKeys()`, `Merge()` | WIRED | `config.LoadGlobal`, `config.LoadRepo`, `config.Merge`, `config.Validate`, `config.ValidateUnknownKeys` all called; `errors.As(err, &ruke)` for `UnknownKeysError` |
| `cmd/root.go` | `cmd/config.go`, `cmd/validate.go` | `rootCmd.AddCommand(configCmd, validateCmd)` | WIRED | Both `configCmd` and `validateCmd` present in `rootCmd.AddCommand(...)` call in root.go |
| `tests/config_cmd_test.go` | zone binary | pre-built binary via `sync.Once`; `exec.Command(binary, args...)` | WIRED | `getZoneBinary()` builds binary at `/workspace/zone`; `runZone()` executes it; all 6 integration tests use this pattern |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CFG-01 | 02-01-PLAN.md | User can create a minimal zone.toml with just `version = 1` and `harness = "claude-code"` | SATISFIED | `TestMinimalConfig` passes; `LoadRepo` sugar handling; `TestConfigVersion` confirms all version paths |
| CFG-02 | 02-01-PLAN.md | User can set global defaults in `~/.config/zone/config.toml` (XDG compliant) | SATISFIED | `GlobalConfigPath()` uses `$XDG_CONFIG_HOME` first; integration tests set `XDG_CONFIG_HOME` and verify global values appear in output with `# global` annotation |
| CFG-03 | 02-02-PLAN.md | Per-repo config overrides global for scalar fields | SATISFIED | `TestScalarOverride` passes; `mergeString` returns repo value when non-empty |
| CFG-04 | 02-02-PLAN.md | List fields merge correctly: packages union, network allow/deny append, extra_args append | SATISFIED | `TestListUnion`, `TestNetworkAllow`, `TestExtraArgsAppend`, `TestListReplace` all pass; merge primitives `mergeUnion`, `mergeAppend`, `mergeReplace` correct |
| CFG-05 | 02-02-PLAN.md | Unknown config keys produce an error with edit-distance suggestions (Levenshtein) | SATISFIED | `TestUnknownKeySuggestion_Close/Far/BareKey/SectionAware` pass; `SuggestKey()` three-pass strategy |
| CFG-06 | 02-02-PLAN.md | Dangerous mount paths are blocked with symlink resolution | SATISFIED | `TestDangerousMount_DockerSocket`, `TestDangerousMount_SymlinkResolution`, `TestDangerousMount_AllCollected` pass; `buildSymlinkChain()` + `filepath.EvalSymlinks` fallback |
| CFG-07 | 02-03-PLAN.md | `zone config` shows merged result with source annotations (global vs repo) | SATISFIED | `TestConfigAnnotatedOutput`, `TestConfigAnnotatedOutput_ListMerge` pass; annotated TOML format with `# repo: zone.toml` / `# global` / `# global (default)` comments |
| CFG-08 | 02-03-PLAN.md | `zone config --json` outputs machine-readable merged config | SATISFIED | `TestConfigJSON`, `TestConfigJSON_Structure` pass; `{value, source}` structure per field |
| CFG-09 | 02-01-PLAN.md | Config schema version field (`version = 1`) is validated on parse | SATISFIED | `TestConfigVersion` passes all three sub-cases: version=0 defaults to 1, version=1 valid, version=2 errors with `ErrVersionMismatch` |
| CFG-19 | 02-02-PLAN.md | Extra mounts default to read-only, require explicit `:rw` for write | SATISFIED | `TestMountReadOnly_NoSuffix/ExplicitRO/ExplicitRW/InvalidPerm` pass; `NormalizeMountPermission()` returns `:ro` when no suffix |

**All 10 requirements satisfied. No orphaned requirements.**

---

### Anti-Patterns Found

No blockers or significant anti-patterns found.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `go.mod` | — | `agnivade/levenshtein` marked as `// indirect` despite being directly imported | Info | `go mod tidy` corrects this automatically; confirmed removed during verification |
| `tests/config_merge_test.go` | 200-203 | `TestHooksAppend` creates unused `global` variable (shadowed by `g`) | Info | Dead code from test refactoring; no behavioral impact, test passes correctly |

---

### Human Verification Required

The following behaviors were verified programmatically but could benefit from human spot-check if desired:

**1. Annotated TOML output format**
**Test:** Run `zone config` in a repo with a zone.toml and global config
**Expected:** TOML output is human-readable with section headers and inline `# source` comments; list fields have comment blocks above the array (not inline, per TOML spec)
**Why human:** Visual formatting quality cannot be fully assessed by grep/assert-contains

**2. `zone validate` exit code 2 behavior in shell scripts**
**Test:** Run `zone validate` on an invalid config in a shell script; check `$?`
**Expected:** Exit code is exactly 2 (not 1)
**Why human:** `os.Exit(2)` bypasses Cobra return handling; confirmed in code but shell behavior worth manual spot-check

---

### Build and Test Summary

| Check | Result |
|-------|--------|
| `go build ./...` | PASS — no errors |
| `go vet ./...` | PASS — no warnings |
| `go test ./tests/ -v -race -count=1` | PASS — 37/37 tests |
| Config merge tests (CFG-01/03/04/09) | PASS — 13 tests |
| Validation tests (CFG-05/06/19) | PASS — 16 tests |
| Integration tests (CFG-07/08) | PASS — 6 tests |

---

### Gaps Summary

No gaps found. All five observable truths are verified, all ten required artifacts exist and are substantive, all key links are wired, and all ten requirements are satisfied by passing tests.

---

_Verified: 2026-03-27_
_Verifier: Claude (gsd-verifier)_
