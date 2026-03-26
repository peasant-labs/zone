---
phase: 1
slug: project-scaffold
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` package (built-in) |
| **Config file** | none — `go test` needs no config file |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go build ./... && go test ./... && goreleaser check` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go build ./... && go test ./... && goreleaser check`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | DX-10 | smoke | `go build ./...` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | DX-10 | smoke | `go test ./...` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | DX-10 | smoke | `goreleaser check` | ❌ W0 | ⬜ pending |
| 01-01-04 | 01 | 1 | DX-10 | smoke | `goreleaser release --snapshot --clean` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go.mod` — `go mod init github.com/peasant-labs/zone` with Go 1.24
- [ ] `.goreleaser.yml` — GoReleaser v2 config for cross-compilation
- [ ] `.golangci.yml` — golangci-lint v2 config
- [ ] All stub `.go` files — ~40 files per spec Section 7
- [ ] `pkg/templates/*.tmpl` — placeholder files for `//go:embed` compilation
- [ ] `Makefile` — build, test, lint, fmt, vet, clean, install targets

*Wave 0 creates all infrastructure — no existing test framework to leverage.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI workflow runs on push | DX-10 | Requires GitHub push trigger | Push commit, verify Actions tab shows green |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
