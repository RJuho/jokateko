+++
title = 'Task and Milestone Timestamps (createdAt, changedAt, targetAt)'
status = 'done'
priority = 'high'
tags = ['backend', 'feature', 'frontend']
summary = 'Standardized RFC3339 UTC timestamps (created_at, changed_at, target_at) across task models, TOML frontmatter parser/writer, SQLite store, REST handlers, MCP tools, and Web UI components with dynamic milestone target timeframe derivations.'
+++

# Task and Milestone Timestamps (createdAt, changedAt, targetAt)

Implement first-class timestamp tracking in Jokateko task frontmatter and milestone aggregations using a consistent RFC3339 UTC format (`_at`).

## Background & Rationale
Git does not track file modification or creation dates, meaning local file metadata is reset on clone or checkout. To maintain Tasks-as-Code determinism, timestamp metadata must be versioned directly inside Markdown frontmatter as the absolute source of truth.

## Requirements & Scope
1. **Frontmatter Schema**:
   - `created_at`: ISO 8601 / RFC3339 UTC string (`YYYY-MM-DDTHH:MM:SSZ`), set once when a task is created.
   - `changed_at`: RFC3339 UTC string, updated automatically whenever a task's content, metadata, checklist, or status is mutated.
   - `target_at`: Optional RFC3339 UTC string representing the target / due date for the task.
2. **Backward Compatibility & Fallback**:
   - For existing tasks without `created_at` or `changed_at`, gracefully fallback to the `YYMMDD` prefix in the task ID or filesystem `mtime` until explicitly updated.
3. **Milestone Target Timeframe**:
   - Milestones dynamically derive their target timeframe (`start` to `end` / `target_at`) from the earliest and latest `target_at` of their associated tasks.
4. **Backend & Tool Integration**:
   - Update `internal/model` Task struct and parser.
   - Update `create_task`, `update_task_content`, `update_task_status`, `update_task_item`, etc., to manage `created_at` and `changed_at`.
   - Add `target_at` to `create_task` and task update MCP tools and CLI.
5. **Web UI Updates**:
   - Display target date badge on task cards (highlighting approaching or overdue dates).
   - Display created / changed / target timestamps in Task Detail modal.
   - Add target date picker/input in task creation and edit modals.
   - Display derived target timeframe in Milestone cards.

## Acceptance Criteria
- [x] Add `created_at`, `changed_at`, and `target_at` to Task model and TOML frontmatter parser/writer
- [x] Automatically set `created_at` on task creation and update `changed_at` on any task mutation
- [x] Provide backward-compatible fallback for legacy tasks without frontmatter timestamps
- [x] Implement dynamic milestone target timeframe derivation based on earliest and latest task target dates
- [x] Expose `target_at` in MCP tools (`create_task`, etc.) and Go CLI commands
- [x] Display `target_at` badges and timestamp metadata in Web UI task cards, detail modals, and milestone cards
- [x] Add comprehensive unit and Playwright E2E test coverage for timestamps and milestone timeframe calculation

## Completion Summary
- **Completed At:** 2026-09-05T16:29:37Z

### What Was Done
1. Extended Task model and TOML frontmatter parser/writer with created_at, changed_at, and target_at fields.
2. Implemented automated timestamp tracking on task creation and mutations across all REST endpoints and Jokateko MCP tools.
3. Added backward-compatible fallback derivation (YYMMDD prefix and file mtime) for legacy tasks without frontmatter timestamps.
4. Added dynamic milestone target timeframe calculation (earliest to latest task target date) in model metrics, SQLite queries, REST endpoints, and MCP milestone tools.
5. Regenerated TypeScript types from Go models and updated Valibot schemas (TaskSchema, MilestoneSchema).
6. Implemented TargetDateBadge in Web UI with approaching/overdue state highlighting, displayed on task cards and in the task detail modal.
7. Added target date picker inputs to CreateTaskModal and TaskEditModal.
8. Added milestone target timeframe displays to MilestoneCards and MilestonesView.
9. Added comprehensive unit tests in Go (model, parser, store, mcp) and Bun (Valibot, components), plus new Playwright E2E tests (26 passing tests).

### Why / Rationale
Tasks-as-Code requires versioned timestamp metadata directly in repository markdown files because Git resets filesystem modification and creation dates on clone or checkout. Standardizing on RFC3339 UTC format preserves deterministic audit trails and enables milestone timeframe planning and delivery tracking for both human developers and AI agents.
