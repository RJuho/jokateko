+++
title = 'Align write-path guards with parse and remove dead config and code'
status = 'backlog'
priority = 'medium'
tags = ['api', 'backend', 'bugfix', 'frontend']
summary = 'Fix the seven gaps found while migrating docs/: make REST and MCP writes enforce the same reference, priority, tag and milestone rules that `jokateko parse` checks, stop force deletes from leaving dangling references, and remove unused config keys, generated types and a dead SSE listener.'
created_at = '2026-10-05T11:25:04Z'
changed_at = '2026-10-05T11:43:03Z'
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
- [ ] (1) Task create/update through the service rejects unknown `milestone` and `dependencies` IDs (not found) and dependency cycles (conflict), for both REST and MCP. Tests cover both
- [ ] (2) `force` deletes clean up references: dependents lose the deleted task ID, and tasks lose the deleted milestone. Each changed file is written atomically and broadcast over SSE. Tests check that `parse` stays clean afterwards
- [ ] (3) Priority is validated against the configured `[[priorities]]`; an unknown value is an error, not silently `medium`. `parse` and the service agree
- [ ] (4) Tag-vocabulary and archived-milestone guards move into `internal/service` and apply to REST and MCP alike. The REST API accepts an explicit `reopen_milestone` (or equivalent), and the UI handles the new error
- [ ] (5) `[mcp] enabled` / `timeout_seconds` are either implemented with tests or removed everywhere (`default.toml`, config structs, `init` template); CFG validation is updated accordingly
- [ ] (6) `generated.ts` is either used in a type check that fails on schema drift, or `cmd/gentypes` and the file are removed (including from `make generate` and `project-structure`)
- [ ] (7) `board.refreshed` is either emitted by the server with a documented trigger, or the client listener is removed
- [ ] Strategies updated so the "known gap" wording is gone: *MCP tool catalog*, *REST API and SSE*, *Go packages and data flow*, *Validation rules and parse output*, *Web UI architecture*
- [ ] `make test`, `make e2e-test` and `jokateko parse` pass

## Notes

### [2026-10-05 11:43 UTC]

Another dead item, found while verifying the README (2026-10-05): `jokateko init` creates an empty `.jokateko/templates/` directory (`cmd/jokateko/init.go:42`, mentioned in its success message), but no config key, loader or UI uses it. Either remove it from `init`, or define what templates are for. Not added as a criterion; handle it together with (5)–(7).
