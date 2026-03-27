# Phase 3: Cache & State - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

`.zone/` directory management: config-hash tracking, file locking with concurrent access protection, build log storage, and Docker artifact ID persistence (image, container, network). Also includes `.gitignore` management. Requirements: CAC-01 through CAC-06.

</domain>

<decisions>
## Implementation Decisions

### Stale lock recovery
- Non-blocking flock attempt per spec — fail immediately if locked (no retry/timeout)
- Write PID to `.zone/.lock.pid` on lock acquire for diagnostics
- Lock contention error includes PID of holding process: "Another zone process (PID 12345) is operating on this repo."
- If PID in `.lock.pid` references a dead process (check `/proc/{pid}` on Linux, `kill -0` on macOS), auto-recover the lock with warning: "Recovered stale lock from dead process (PID 12345)."
- `zone clean` warns but proceeds even if lock is held: "Warning: another zone process (PID 12345) may be running. Cleaning anyway."
- Exit code 5 for lock contention (per spec)

### Cache invalidation cleanup
- Old Docker images are left in Docker's local store after rebuild — just update `.zone/image_id`
- No auto-pruning of old images; user can `docker image prune` or `zone destroy`
- If cached `image_id` references a pruned/deleted image, silently rebuild (detected via `ImageInspect`); log at verbose level only: "Cached image not found, rebuilding"
- All cache file writes (config.hash, image_id, container_id, network_id) use atomic write: write to `.zone/.tmp-{name}`, then `os.Rename`
- `zone destroy` requires `--yes`/`-y` flag to skip interactive confirmation; without it, print what will be removed and ask for confirmation

### Gitignore management
- If no `.gitignore` exists, create a minimal one with just `.zone/`
- Find git root via `git rev-parse --show-toplevel` and modify that `.gitignore` (correct for monorepos and subdirectory invocations)
- In monorepos/subdirectories, use relative path from git root: e.g., `subdir/.zone/` (not `**/.zone/` wildcard)
- Idempotent: exact string check — skip if entry already present; don't attempt to parse gitignore glob semantics

### Build log retention
- Last build log only (`last_build.log`), overwritten each build — matches spec
- Keep partial log on build failure — the partial output is the most useful debugging artifact
- Tee build output to both terminal and log file simultaneously (user sees progress live)
- Prepend brief metadata header to log: `# zone build | {timestamp} | config hash: {hash} | zone {version}`

### Claude's Discretion
- Internal cache directory management (ensure-dir helpers, file permission modes)
- Hash computation serialization details (deterministic JSON encoding of MergedConfig)
- Exact flock syscall usage (syscall.Flock vs golang.org/x/sys)
- Build log tee implementation (io.MultiWriter or similar)
- Error message wording for edge cases not covered above

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Cache directory & locking
- `zone-spec.md` section 6 — Full `.zone/` directory structure, all file names and purposes
- `zone-spec.md` section 6.1 — Concurrent access protection: flock semantics, non-blocking, exit code 5
- `zone-spec.md` section 6.2 — Cache hash computation: SHA256 of merged config JSON + Dockerfile template + entrypoint template + Zone version
- `zone-spec.md` section 6.3 — Gitignore management rules

### Launch flow (cache consumer)
- `zone-spec.md` section 3.2 — Launch lifecycle steps 0-4: lock acquire, container state check, hash comparison, image verification, cache file updates

### Error handling
- `zone-spec.md` section 8 — Error handling convention, exit code 5 = cache error

### Project structure
- `zone-spec.md` section 7 — Package layout: `internal/cache/` with cache.go, hash.go, lock.go

### Project context
- `.planning/PROJECT.md` — Tech stack constraints, key decisions
- `.planning/REQUIREMENTS.md` — CAC-01 through CAC-06 requirement details

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/cache/cache.go`: Package stub — needs full `.zone/` directory management implementation
- `internal/cache/hash.go`: Package stub — needs SHA256 hash computation using MergedConfig + templates + version
- `internal/cache/lock.go`: Package stub — needs flock-based locking with PID tracking
- `internal/config/types.go`: `MergedConfig` struct — consumed by hash computation (serialize to JSON for hashing)
- `internal/config/config.go`: Config loading functions — cache uses merged config output
- `tests/hash_test.go`: Test stub — needs cache hash tests

### Established Patterns
- Cobra `RunE` pattern with `fmt.Errorf("not implemented")` stubs (Phase 1)
- Error collection pattern from Phase 2 validation — collect all issues, report together
- Config types use `*bool` for nullable fields, typed structs throughout

### Integration Points
- `cmd/clean.go`: Cobra stub — will call into `internal/cache/` for `.zone/` cleanup
- `cmd/destroy.go`: Cobra stub — will call cache cleanup as part of full teardown (Phase 6 wires the full destroy)
- `internal/config/` package provides `MergedConfig` that feeds into hash computation
- Phase 4 (templates) and Phase 6 (lifecycle) will be primary consumers of this cache layer
- Build log storage prepares for `zone logs --build` command (wired in Phase 8)

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the spec and decisions above. The spec is prescriptive for cache structure — Section 6 includes exact directory layout, Go code for hash computation, and flock semantics.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-cache-state*
*Context gathered: 2026-03-27*
