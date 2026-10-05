+++
title = 'Go test coverage T1: service, mcp and watcher'
status = 'done'
priority = 'high'
tags = ['backend', 'testing']
summary = 'Direct tests for the core mutation layer raised coverage to service 95.5%, mcp 95.3%, watcher 94.4% (from 59.1/81.1/65.7).'
created_at = '2026-10-05T05:47:59Z'
changed_at = '2026-10-05T06:11:39Z'
+++

## Context
Baseline measured 2026-10-05: per-package total 77.2%, cross-package (`-coverpkg`) 80.3%. Coverage for `service` comes mostly from the mcp and server tests, so its own tests cover only 59.1%. Goal for the whole effort: every `internal/*` package ≥85% and the total ≥85%. Generated sqlc code and `main()` are excluded.

| pkg | own % | target |
|---|---|---|
| internal/service | 59.1 | ≥85 |
| internal/mcp | 81.1 | ≥88 |
| internal/watcher | 65.7 | ≥85 |

Test helpers to reuse: `newTestService` and `recordingNotifier` (`internal/service/service_test.go`), `setupTestMCP` and `callToolJSON[Out]` (`internal/mcp/tools_task_test.go`).

## Acceptance Criteria
- [x] service/tasks.go: `CompleteTask` (summary recorded, downstream unblocked, already done), `SetTaskTarget` + `NormalizeTargetAt` (RFC3339, YYYY-MM-DD, invalid, clear), `AddTaskNote`, `RemoveTaskDependency` (dependency that doesn't exist), `GetTask` not found, `defaultTaskStatus` branches
- [x] service/entities.go: `GetMilestone`, `UpdateMilestone`, `DeleteMilestone` (refused while tasks are assigned), `ParseMilestoneStatus` table test, `UpdateStrategy`, `GetGlossaryTerm`, `UpdateGlossaryTerm`, `DeleteGlossaryTerm`, `wrapGet` error mapping, `writeEntity`/`removeEntityFile` filesystem errors
- [x] service/service.go: `CheckTags` (enforced vs. not), `Error()`, accessors (`Config`, `WorkspaceDir`, `Dirs`, `Store`), `New` error path
- [x] Each service test asserts the notifier event and the on-disk file contents
- [x] mcp/tools_search.go: `search_milestones`, `search_strategies`, `search_glossary` (hit, miss, tag filter); `resolveLimit` (0, negative, over the max)
- [x] mcp/tools_task.go: `set_task_target` (valid, invalid date, unknown ID); `update_task_content` error branches; `add_task_note` and `get_task` not found
- [x] mcp/tools_milestone.go `update_milestone` branches (status, timeframe, reopen); glossary/strategy delete and lookup not-found paths
- [x] mcp: `HTTPHandler` round trip over httptest (initialize + tools/list); `priorityScore` table test; `resourceStrategyTiers`
- [x] watcher: `StartPipeline` end to end (write file → ingest + broadcast → cancel context → clean shutdown) using bounded polling, no sleep
- [x] watcher: `New` with a missing dir, `Errors()`, `isInDir` table test (prefix collision such as `tasks` vs `tasks2`)
- [x] `go test ./...` passes; per-package coverage numbers recorded in a task note

## Conventions
Standard `testing` only, no new dependencies. Use `t.TempDir()`, `t.Context()`, table tests with `t.Run`. Load the `use-modern-go` skill first. Never touch `.jokateko/` from tests.

## Notes

### [2026-10-05 05:58 UTC]

Coverage per package (`CGO_ENABLED=0 go test -cover`), before → after:
- internal/service: 59.1% → 95.5%
- internal/mcp: 81.1% → 95.3%
- internal/watcher: 65.7% → 93.9%

New test files (test-only, no production changes): `internal/service/tasks_test.go`, `internal/service/entities_test.go`, `internal/mcp/internal_test.go` (package mcp: priorityScore, resolveLimit, requireID, isArchived, every mutating tool with allow_mutations=false, closed-store error paths), `internal/mcp/coverage_test.go`, `internal/watcher/isindir_internal_test.go`, `internal/watcher/pipeline_test.go`.

Caveats:
- `service.New` has no error return. "New error path" is covered as the nil-config/nil-notifier fallback.
- The "notifier event + on-disk file" checks apply to every mutating service test. Pure-function tests (ParseMilestoneStatus, wrapGet, CheckTags, NormalizeTargetAt, accessors) have no event or file to check.
- `-race` could not run in this devcontainer: no C compiler (gcc) is installed, and the race detector needs cgo. Flakiness was checked instead with `-count=3`, `-count=10 -shuffle=on` and `-count=20 -cpu=1,4` on the watcher pipeline tests. All runs passed.
- StartPipeline clean shutdown is checked by polling until `runtime.NumGoroutine()` drops back to its pre-start baseline. There is no sleep.

Bugs found: none blocking. One minor latent edge case: `Pipeline.isInDir` uses `strings.HasPrefix(rel, "..")`, so a file named like `..foo.md` directly inside an entity dir would be treated as outside it. This is not reachable in practice, because IDs cannot start with `.` and dotfiles are ignored by the watcher and by ProcessAll. No test asserts it.

Item 11 ("go test ./... passes") is left for the coordinator.

### [2026-10-05 06:03 UTC]

Coordinator validation (2026-10-05): only *_test.go files were added in scope. `CGO_ENABLED=0 go test ./...` passes, including test/chaos. `go test -count=5 -shuffle=on -cpu=1,4 ./cmd/... ./internal/...` passes, so no flaky tests were seen. gofmt and go vet are clean. Coverage: service 95.5%, mcp 95.3%, watcher 94.4%. `-race` was not run because there is no gcc in the devcontainer. `make cover` total: 93.2%.

## Completion Summary
- **Completed At:** 2026-10-05T06:11:39Z

### What Was Done
- New internal/service/tasks_test.go and entities_test.go: CompleteTask, SetTaskTarget/NormalizeTargetAt, AddTaskNote, RemoveTaskDependency, milestone/strategy/glossary get/update/delete (including the delete guards), wrapGet error mapping, writeEntity/removeEntityFile filesystem errors, CheckTags, accessors, the New fallbacks. Tests that write something assert the notifier event and the file on disk.
- New internal/mcp/internal_test.go and coverage_test.go: milestone/strategy/glossary search tools, resolveLimit, set_task_target, update_task_content error branches, update_milestone branches, not-found paths, every write tool with mutations turned off, closed-store errors, HTTPHandler round trip, priorityScore, resourceStrategyTiers.
- New internal/watcher/pipeline_test.go and isindir_internal_test.go: StartPipeline end to end with bounded polling and a check that its goroutines exit; New with a missing dir; Errors(); an isInDir table test.

### Why / Rationale
service is the single mutation path for REST and MCP, but its own tests covered only 59%. Its behaviour was tested only indirectly through the handlers. Direct tests pin the business rules (guards, notifications, atomic writes) where they live. The tests use bounded polling instead of sleeps to stay deterministic; verified with -count=5 -shuffle=on -cpu=1,4.
