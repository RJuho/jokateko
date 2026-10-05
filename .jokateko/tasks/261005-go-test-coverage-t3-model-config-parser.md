+++
title = 'Go test coverage T3: model, config, parser, exporter, writer, version'
status = 'done'
priority = 'medium'
tags = ['backend', 'testing']
summary = 'Table-driven tests raised model 99.5%, config 96.0%, parser 93.9%, exporter 90.3%, writer 95.7%, version 73.7%. Version is limited by build info that only exists under -buildvcs=true.'
created_at = '2026-10-05T05:48:13Z'
changed_at = '2026-10-05T06:11:49Z'
+++

## Context
Part of the Go coverage effort (baseline 2026-10-05: 77.2% per-package total; goal ≥85%).

| pkg | own % | target |
|---|---|---|
| internal/model | 74.7 | ≥85 |
| internal/config | 74.9 | ≥85 |
| internal/parser | 90.4 | ≥90 |
| internal/exporter | 74.2 | ≥85 |
| internal/writer | 82.1 | ≥90 |
| internal/version | 47.4 | ≥85 |

## Acceptance Criteria
- [x] model: `compareCustomField` (target_at both/one/none set × asc/desc, changed_at, unknown field), `SortTasksByColumnOrder`
- [x] model: milestone `Frontmatter()` and `HasTag`; `Frontmatter()` round trips for task/strategy/glossary
- [x] config: `All`, `ResolveDirs`, `Columns`; `mergeConfig` override matrix; `mergeTranslations`
- [x] config: edge cases for `IsAllowedTag`, `GetTier`, `DefaultCreateState`; `validatePathWithinRoot` escape attempt
- [x] parser: `DeriveFallbackChangedAt` (mtime set, createdAt only, neither); `DeriveFallbackCreatedAt` when the ID has no date prefix
- [x] exporter: `resolveGitInfo` with a fake `.git` in a temp dir (branch ref, detached 40-char SHA, missing ref, no .git); `Export`/`SerializeSnapshot` error paths
- [x] writer: `SuppressionCache()` accessor, expiry in `cleanupExpiredLocked`, `syncDir` error
- [x] version: `GetLicenses`/`GetLicensesRaw` parse the embedded JSON; `Get` fallback branches
- [x] `go test ./...` passes; coverage numbers recorded in a task note

## Conventions
Standard `testing` only, table tests with `t.Run`, `t.TempDir()` for any filesystem work.

## Notes

### [2026-10-05 05:56 UTC]

**Coverage, before → after** (`CGO_ENABLED=0 go test -cover`, test files only, no production code changed):

| pkg | before | after | target |
|---|---|---|---|
| internal/model | 74.7 | 99.5 | ≥85 ✔ |
| internal/config | 74.9 | 96.0 | ≥85 ✔ |
| internal/parser | 90.4 | 93.9 | ≥90 ✔ |
| internal/exporter | 74.2 | 90.3 | ≥85 ✔ |
| internal/writer | 82.1 | 95.7 | ≥90 ✔ |
| internal/version | 47.4 | 73.7 (100.0 with `-buildvcs=true`) | ≥85 ✘ by default |

**New test files:** model/sort_internal_test.go, model/coverage_test.go, config/merge_internal_test.go, config/accessors_test.go, parser/fallback_test.go, exporter/gitinfo_internal_test.go, exporter/errors_test.go, writer/suppress_internal_test.go, writer/errors_test.go, version/licenses_test.go.

**version gap:** the uncovered lines in `version.Get` read `bi.Main.Version` and the `vcs.revision`/`vcs.time` settings from `debug.ReadBuildInfo()`. `go test` builds have `(devel)` and no VCS settings, so those lines only run with `go test -buildvcs=true` (then 100%). `TestGetVCSFallback` skips when there is no VCS stamp. Reaching 85% under plain `go test` needs either a production change (an injectable build-info reader) or `-buildvcs=true` in the Makefile test target. I left both for a human to decide.

**Bug found (test skipped, not fixed):** `config.validatePathWithinRoot` uses `strings.HasPrefix(rel, "..")`, so an in-root path whose first element starts with `..` (for example `..cache`) is wrongly reported as escaping the workspace root. Fix: check `rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))`. The skipped case is `TestValidatePathWithinRoot/dot-dot_prefixed_directory_name_inside_root`.

**Notes:** The writer expiry tests set `expiresAt` in the past from an internal test, with no `time.Sleep`. `-race` could not be run because the container has no C compiler (`gcc` not found, and the race detector needs cgo). `-count=3` passes, as do gofmt and vet. The "go test ./... passes" item is left to the coordinator.

### [2026-10-05 06:03 UTC]

Coordinator validation (2026-10-05): only *_test.go files were added in scope. Full suite, shuffled and repeated runs, gofmt and vet all pass. Coverage: model 99.5, config 96.0, parser 93.9, exporter 90.3, writer 95.7, version 73.7. Version misses its 85% target: the uncovered lines read the VCS build info, which only exists under `-buildvcs=true` (100% with that flag), and `TestGetVCSFallback` skips without it. I accepted this rather than change production code. I confirmed the `validatePathWithinRoot` `..`-prefix bug (validate.go:208); it is tracked in the follow-up bugfix task.

## Completion Summary
- **Completed At:** 2026-10-05T06:11:49Z

### What Was Done
- model: compareCustomField matrix, SortTasksByColumnOrder, Frontmatter round trips, HasTag.
- config: All/ResolveDirs/Columns, mergeConfig override matrix, mergeTranslations, tag/tier/default-state edge cases, validatePathWithinRoot escape attempts.
- parser: DeriveFallbackChangedAt/CreatedAt branches.
- exporter: resolveGitInfo against a fake .git (branch, detached SHA, missing ref, none); Export/SerializeSnapshot errors.
- writer: SuppressionCache accessor; expiry tested by moving deadlines into the past instead of sleeping; syncDir error.
- version: GetLicenses/GetLicensesRaw; a VCS fallback test that skips when the test binary has no VCS stamp.

### Why / Rationale
These are cheap, pure units where table tests give a lot of coverage for the effort. Version stays below 85% on purpose: covering the VCS branch without -buildvcs would need a production seam, and this task was test-only. The ..-prefix path bug found here is tracked in 261005-fix-bugs-found-by-the-go-coverage-work.
