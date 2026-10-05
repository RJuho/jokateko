+++
title = 'Align write-path guards with parse and remove dead config and code'
status = 'done'
priority = 'medium'
tags = ['api', 'backend', 'bugfix', 'frontend']
summary = 'REST and MCP writes now go through the same service guards that `jokateko parse` checks: references, cycles, configured priorities, tag vocabulary and archived milestones. Force deletes clean up references. The unused `[mcp] timeout_seconds` key and the `board.refreshed` listener are gone, `[mcp] enabled` now works, and generated.ts is checked against the Valibot schemas at type level.'
created_at = '2026-10-05T11:25:04Z'
changed_at = '2026-10-05T14:10:20Z'
+++

## Context
These gaps were found on 2026-10-05 while checking the old `docs/` against the code (task `261005-migrate-docs-into-strategies-and-glossar`). The strategies currently describe each one as a known gap: *MCP tool catalog*, *REST API and SSE*, *Go packages and data flow*, *Validation rules and parse output*, *Web UI architecture*. Update those strategies when a gap is fixed.

The principle (from *Go packages and data flow*): guards that should apply to every client belong in `internal/service`, not in `internal/mcp` or `internal/server`. Nothing written through the service should fail `jokateko parse`.

## Findings

### Write-path validation (service)
1. **References not verified on create and update.** MCP `create_task` / `update_task_content` (and REST `POST`/`PUT /api/tasks`) accept a `milestone` and `dependencies` that don't exist, and run no cycle check. Only `AddTaskDependency` validates. `parse` later reports `TSK-006` / `TSK-007` / `DAG-001`.
2. **`force` deletes leave dangling references.** `DeleteTask(force)` does not remove the ID from dependents' `dependencies`. `DeleteMilestone(force)` does not clear the tasks' `milestone`. Both later fail `parse`.
3. **Priority is checked against built-in IDs.** `CreateTask` uses `model.Priority.IsValid()` (the four defaults) and silently maps unknown values to `medium`, ignoring `[[priorities]]`. `parse` (`TSK-005`) uses the configured list.
4. **Tag and archived-milestone guards are MCP-only.** `CheckTags` and `checkMilestoneOpen` run in `internal/mcp`, so REST writes bypass the `[tags] enforce_allowed` vocabulary and the `reopen_milestone` rule.

### Dead config and code
5. **`[mcp] enabled` and `[mcp] timeout_seconds`** are loaded in `internal/config/load.go` but nothing reads them. Either implement them (`enabled = false` → no `/api/mcp`, and `jokateko mcp` refuses to start; a timeout per tool call) or remove them from `default.toml`, the config structs and the `init` template.
6. **`web/src/types/generated.ts`** (output of `cmd/gentypes`, run by `make generate`) is imported nowhere; the Valibot schemas in `web/src/schemas/models.ts` are hand-maintained. Either use it (for example, a type-level test that the schema output types match the generated interfaces) or drop the generator and the file.
7. **Dead SSE listener.** `web/src/state/sse.ts` handles `board.refreshed`, but the server never emits it. Remove the listener, or emit it where a full resync is needed (for example after a burst of watcher events or a dropped-event overflow).

## Acceptance Criteria
- [x] (1) Task create/update through the service rejects unknown `milestone` and `dependencies` IDs (not found) and dependency cycles (conflict), for both REST and MCP. Tests cover both
- [x] (2) `force` deletes clean up references: dependents lose the deleted task ID, and tasks lose the deleted milestone. Each changed file is written atomically and broadcast over SSE. Tests check that `parse` stays clean afterwards
- [x] (3) Priority is validated against the configured `[[priorities]]`; an unknown value is an error, not silently `medium`. `parse` and the service agree
- [x] (4) Tag-vocabulary and archived-milestone guards move into `internal/service` and apply to REST and MCP alike. The REST API accepts an explicit `reopen_milestone` (or equivalent), and the UI handles the new error
- [x] (5) `[mcp] enabled` / `timeout_seconds` are either implemented with tests or removed everywhere (`default.toml`, config structs, `init` template); CFG validation is updated accordingly
- [x] (6) `generated.ts` is either used in a type check that fails on schema drift, or `cmd/gentypes` and the file are removed (including from `make generate` and `project-structure`)
- [x] (7) `board.refreshed` is either emitted by the server with a documented trigger, or the client listener is removed
- [x] Strategies updated so the "known gap" wording is gone: *MCP tool catalog*, *REST API and SSE*, *Go packages and data flow*, *Validation rules and parse output*, *Web UI architecture*
- [x] `make test`, `make e2e-test` and `jokateko parse` pass

## Notes

### [2026-10-05 11:43 UTC]

Another dead item, found while verifying the README (2026-10-05): `jokateko init` creates an empty `.jokateko/templates/` directory (`cmd/jokateko/init.go:42`, mentioned in its success message), but no config key, loader or UI uses it. Either remove it from `init`, or define what templates are for. Not added as a criterion; handle it together with (5)–(7).

### [2026-10-05 14:05 UTC]

Implementation summary (for review):

- **Service guards** (`internal/service/tasks.go`): `checkTaskLocked` runs on create and inside `updateTaskLocked`, so every `UpdateTask` caller is covered. On update it checks only the fields that **changed**: added tags, a new priority, a new milestone and added dependencies. Files with legacy problems therefore stay editable. A new `PatchTask(TaskPatch)` replaces the patch logic that was duplicated in the REST and MCP handlers. An empty title or summary is now an error; MCP used to ignore it silently.
- **Archived milestone**: `ErrConflict` with code `milestone_archived` (`service.ErrorCode`). REST returns `{"error","code"}`. `utils/taskApi.ts` `saveTask` asks with `confirm()` and retries with `reopen_milestone: true`. The new i18n key is `task_milestone_reopen_confirm`.
- **Decisions agreed with the user**: (5) implement `[mcp] enabled` (no `/api/mcp` handler; `jokateko mcp` exits 1) and remove `timeout_seconds`; stale keys are ignored when loading. (6) `web/src/types/drift.ts` checks that the field names are equal and that the generated type is assignable to the schema input; `make typecheck` now runs first in `make test`. (7) remove the `board.refreshed` listener; the gentypes SSE union now lists the strategy and glossary events.
- **Found by the drift check**: the About modal read `build.go_version`/`build.platform`, which Go never sent. Added both to `model.BuildConfig` and filled them in from `version.Get()` in the exporter.
- **Fixed in passing**: the edit modal sent `milestone: null` to clear a milestone, which the server treats as "unchanged". It now sends `''`.
- `jokateko init` no longer creates `.jokateko/templates/`. This repo's own empty `.jokateko/templates/` directory is untracked by git and was left in place, because `.jokateko/` must not be edited directly. Delete it by hand if you want.
- **E2E**: `tests/e2e/milestone-reopen.spec.ts`, plus `data-testid` on both milestone selects. Spec files must not contain TypeScript annotations: the runner fails with an unexplained "N errors building".

## Completion Summary
- **Completed At:** 2026-10-05T14:10:20Z

### What Was Done
- `internal/service`: `checkTaskLocked` runs on create and on every update, checking only the fields that changed: added tags, priority against `[[priorities]]`, milestone existence plus the archived guard (`ErrConflict` with code `milestone_archived`), and dependency existence, self-dependency and cycles. `PatchTask`/`TaskPatch` replaces the patch logic that REST and MCP each had. Milestone, strategy and glossary create/update now check the tag vocabulary. `DeleteTask(force)` and `DeleteMilestone(force)` rewrite the referencing tasks before deleting.
- REST accepts `reopen_milestone` and returns `code` in errors. MCP-only guards were removed from `internal/mcp`. The UI's `utils/taskApi.ts` asks with `confirm()` and retries; the edit modal now clears a milestone with `''`.
- `[mcp] enabled` is implemented in `serve` and `mcp`; `timeout_seconds` is removed. `web/src/types/drift.ts` plus `make typecheck` runs first in `make test`. That check found `go_version` and `platform` missing from `model.BuildConfig`; both were added. The SSE union in gentypes was fixed and the `board.refreshed` listener removed. `init` no longer creates `templates/`.
- Tests: service guard tests with parse-clean checks after force deletes, REST, MCP and CLI tests, and the e2e spec `milestone-reopen.spec.ts`. Seven strategies were updated.

### Why / Rationale
The *Go packages and data flow* rule says guards that apply to every client belong in `internal/service`, and nothing written through the service should fail `parse`. Checking only the fields that change keeps files with older problems editable while new writes stay valid. A machine-readable error code lets the UI handle the archived-milestone case without parsing message text. Dead config and code were removed, or wired up where the user wanted the feature: `[mcp] enabled` as a security switch, and the drift check to make gentypes useful.
