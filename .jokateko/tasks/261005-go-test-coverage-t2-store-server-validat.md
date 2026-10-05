+++
title = 'Go test coverage T2: store, server, validator and proxy'
status = 'done'
priority = 'medium'
tags = ['backend', 'testing']
summary = 'Error-path and handler tests raised coverage to store 93.6%, server 98.8%, validator 94.2%, proxy 92.1%, and found the FTS5 unbalanced-quote 500 and a validator index bug.'
created_at = '2026-10-05T05:48:06Z'
changed_at = '2026-10-05T06:11:44Z'
+++

## Context
Part of the Go coverage effort (baseline 2026-10-05: 77.2% per-package total; goal ≥85% total and per `internal/*` package).

| pkg | own % | target |
|---|---|---|
| internal/store | 79.0 | ≥85 (generated `queries.sql.go` excluded) |
| internal/server | 88.6 | ≥90 |
| internal/validator | 76.9 | ≥90 |
| internal/proxy | 79.6 | ≥85 |

Helpers to reuse: `setupTestServer` (`internal/server/server_test.go`), `startLiveServer` (`internal/server/security_test.go`), the fake-daemon harness in `internal/proxy/proxy_test.go`.

## Acceptance Criteria
- [x] store: error branches of `UpsertTask/Milestone/Strategy/GlossaryTerm` and `Delete*` (missing ID, closed DB)
- [x] store: `UpdateTaskCriteria`, `GetEntityTags`, `Lock`/`Unlock`
- [x] store: `formatTimeframeDate`, `deriveMilestoneTimeframe`, `formatFTS5Query` edge cases (quotes, operators, empty), all as table tests
- [x] server: `GET /api/milestones/{id}`, `/api/strategies/{id}`, `/api/glossary/{id}` (200, 404, invalid ID)
- [x] server: 4xx/5xx paths in the list and delete handlers; `handleUpdateTask` validation errors
- [x] server: `Shutdown`, `Port` on a live server; SSE `Broadcast` with no subscribers
- [x] validator: `FormatReport` (empty, warnings only, mixed, ordering), `ErrorCount`/`WarningCount`/`HasErrors`, `Diagnostic.String`
- [x] validator: `ValidateMilestone` and `ValidateGlossaryTerm` rule branches
- [x] proxy: `replayHandshake` failure and partial-replay paths; `switchToStandalone` when the daemon dies
- [x] `go test ./...` passes; coverage numbers recorded in a task note

## Conventions
Standard `testing` only, no new dependencies. No `time.Sleep` for synchronisation. Never touch `.jokateko/` from tests.

## Notes

### [2026-10-05 06:01 UTC]

Coverage, before → after (`CGO_ENABLED=0 go test -cover`):
- internal/store: 79.0% → 93.6% (hand-written files 95.8%, generated queries.sql.go 85.0%)
- internal/server: 88.6% → 98.8%
- internal/validator: 76.9% → 94.2%
- internal/proxy: 79.6% → 92.1%

New test files (no production code changed): internal/store/failure_test.go, internal/server/coverage_test.go, internal/server/internal_test.go, internal/validator/coverage_test.go, internal/proxy/coverage_test.go, internal/proxy/internal_test.go. Store failures are injected through SQLite triggers or dropped tables. FTS5 virtual tables can't carry triggers, so tests swap in a plain table to reach the FTS insert error paths.

Verification: all four packages pass with `-count=3`. `gofmt -l` prints nothing and `go vet` is clean. I could not run `-race` because this container has no C toolchain (CGO_ENABLED=0, no gcc), so it still needs a run on a machine that has one.

Bug found (not fixed, test skipped as `TestSearchUnbalancedQuotePassthrough`): `formatFTS5Query` passes any input that starts and ends with `"` through unchanged. A lone `"` or `"a "b"` therefore reaches FTS5 unescaped, and SearchAll returns `SQL logic error: unterminated string` (HTTP 500 from /api/search).

Other findings:
- The proxy's handshake cache (`inspectClientMessage` / `replayHandshake`) never fires with the current go-sdk v1.8.0 client. That client opens with `server/discover` (protocol 2026-07-28) instead of `initialize`, so nothing gets cached. Replay is covered by unit tests with a fake connection and by a raw-JSON-RPC failover test that sends `initialize` itself.
- Code that can't be reached (left uncovered): MLS-003 and GLS-002 required-field checks in rules.go, because the parser rejects a missing title or summary first, so those cases surface as MLS-002 and GLS-001. Also the TSK-002 non-`+++` delimiter branch, the TSK-003 checks, and the config.Validate multi-error branch in ValidateWorkspace (config.Load already validates). Also in validator.go: `len(msg) >= 7 && msg[7] == ':'` would panic on a message exactly 7 characters long. Validate never produces one today, but the check should be `>= 8`.

### [2026-10-05 06:03 UTC]

Coordinator validation (2026-10-05): only *_test.go files were added in scope. Full suite, shuffled and repeated runs (`-count=5 -shuffle=on -cpu=1,4`), gofmt and vet all pass. Coverage: store 93.6%, server 98.8%, validator 94.2%, proxy 92.1%. I confirmed both reported bugs in the source (`formatFTS5Query` passes unbalanced quotes straight through; validator.go:150 `len(msg) >= 7 && msg[7]`). They are tracked in the follow-up bugfix task. `TestSearchUnbalancedQuotePassthrough` is skipped until that fix lands. `-race` was not run because there is no gcc.

## Completion Summary
- **Completed At:** 2026-10-05T06:11:44Z

### What Was Done
- internal/store/failure_test.go: SQL errors forced with SQLite triggers or dropped tables (the FTS5 table is swapped for a plain table, since FTS5 can't take triggers) cover every Upsert/Delete error branch. Also UpdateTaskCriteria, GetEntityTags, Lock/Unlock, table tests for the timeframe and formatFTS5Query helpers.
- internal/server/coverage_test.go and internal_test.go: single-entity GET handlers (200/404/invalid ID), 4xx/5xx in the list and delete handlers, handleUpdateTask validation, Shutdown/Port on a live server, Broadcast with no subscribers.
- internal/validator/coverage_test.go: FormatReport variants, counters, Diagnostic.String, milestone and glossary rule branches.
- internal/proxy/coverage_test.go and internal_test.go: replayHandshake failure and partial-replay paths, failover when a live daemon is killed.

### Why / Rationale
Persistence and API error paths are where silent data loss or wrong HTTP status codes would hide. Forcing failures inside SQLite tests the real error handling without mocking the database. The bugs found are tracked in 261005-fix-bugs-found-by-the-go-coverage-work rather than fixed here, so this task stayed test-only.
