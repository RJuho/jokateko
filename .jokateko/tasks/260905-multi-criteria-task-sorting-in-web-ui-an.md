+++
title = 'Multi-Criteria Task Sorting in Web UI and MCP'
status = 'done'
priority = 'medium'
tags = ['backend', 'feature', 'frontend']
summary = 'Implemented configurable multi-criteria task sorting across Kanban columns and MCP endpoints with interactive per-column dropdown controls and done-state recency sorting.'
dependencies = ['260904-task-and-milestone-timestamps-createdat-']
created_at = '2026-09-05T17:02:29Z'
changed_at = '2026-09-06T06:17:13Z'
+++

# Multi-Criteria Task Sorting in Web UI and MCP

Implement configurable, multi-tier task sorting across Kanban columns and Model Context Protocol (MCP) endpoints, incorporating priority tiers, target dates, and modification timestamps.

## Background & Rationale
Tasks in different Kanban workflow stages require distinct prioritization semantics. In backlog and completed states, recency of modification (`changed_at`) reflects active focus and recent accomplishments. In active execution states (`ready`, `in_progress`, `in_review`), proximity to target deadline (`target_at`) is paramount to ensure approaching deadlines are not overlooked. Web UI users and AI agents should experience consistent ordering aligned with workflow states.

## Requirements & Scope
1. **Default Sorting Model**:
   - **Primary Tier**: Priority (`critical` > `high` > `medium` > `low`).
   - **Secondary Tier (State-Specific)**:
     - `backlog` and `done`: `changed_at` descending (newest to oldest).
     - `ready`, `in_progress`, and `in_review`: `target_at` ascending (soonest deadline first). If `target_at` is empty or legacy, fallback to `changed_at` descending.
   - **Tertiary Tier**: `id` ascending for deterministic tie-breaking.
2. **Configurability (`config.toml`)**:
   - Column configuration in `[[board.columns]]` can optionally define custom `sort_by` and `sort_direction` rules, falling back to the default workflow rules.
3. **Web UI Sort Control**:
   - Add a "Sort By" dropdown button in `FilterBar.tsx` next to tag filters.
   - Options include:
     - *Default (Workflow State Sorting)*
     - *Priority (Critical to Low)*
     - *Target Date (Soonest First)*
     - *Recently Changed (`changed_at`)*
     - *Created Date (`created_at`)*
     - *Alphabetical (Title A-Z)*
   - Manage active sorting via a Preact signal in `store.ts` and compute reactive column ordering in `columnTasks`.
4. **MCP Consistency**:
   - MCP `list_tasks` and `get_board_state` handlers return tasks sorted by the configured state-specific defaults, ensuring agent queries mirror the human board view.

## Acceptance Criteria
- [x] Implement multi-criteria sorting comparator supporting priority tiers, target_at (ascending), and changed_at (descending)
- [x] Support optional per-column sort configurations in `[[board.columns]]` with fallback defaults
- [x] Add Sort dropdown button to Web UI `FilterBar` with interactive sort mode selection
- [x] Update Preact store `columnTasks` computed signal to reactively sort tasks per column
- [x] Align MCP `list_tasks` and `get_board_state` output ordering with column default sorting
- [x] Add comprehensive unit tests in Go and Bun for sorting determinism and edge cases (missing target_at, ties)
- [x] Add Playwright E2E tests validating Sort By dropdown interactions and board card ordering

## Notes

### [2026-09-06 05:53 UTC]

Implemented multi-criteria task sorting across backend Go models, configuration, MCP endpoints (`list_tasks`, `get_board_state`), and Preact Web UI (`FilterBar` Sort By dropdown and `columnTasks` computed signal). All Go unit tests, Bun unit tests, and Playwright E2E tests are passing. Built binary and copied to `/home/bun/.local/bin/jokateko`. Moved to In Review for human verification.

### [2026-09-06 06:09 UTC]

Updated column sorting UX per user feedback:
- Relocated sort options from FilterBar to individual Column header pills (located on the right side next to the '+' button).
- Enabled independent per-column sorting modes (default, priority, target_at, changed_at, created_at, title), managed in Preact store and reset via filter bar reset.
- Fixed dropdown opening and clipping by positioning dropdown on column pill outside overflow scroll containers.
- Updated internal/config/default.toml with sort_by, sort_direction, and full translations configuration.
- Verified with 4 dedicated Playwright E2E tests in tests/e2e/task-sorting.spec.ts (all 32 E2E tests passing), Bun unit tests (50/50 passing), and Go tests/lint (all passing).
- Rebuilt executable and installed to /home/bun/.local/bin/jokateko. Task remains in in_review (In Preview) for human verification.

### [2026-09-06 06:15 UTC]

Updated default sorting for 'done' state:
- For 'done' state, default sorting now directly sorts by Recently Changed (`changed_at` descending) without applying the priority tier, as priority does not matter anymore once tasks are completed.
- Updated Go backend comparator `CompareTasksForColumn` in `internal/model/sort.go` and Web UI comparator `compareTasks` in `web/src/utils/sort.ts`.
- Updated `internal/config/default.toml` documentation to reflect the done state sorting rule.
- Added comprehensive unit tests in Go (`internal/model/sort_test.go`), Bun (`web/src/utils/sort.test.ts`), and Playwright E2E (`tests/e2e/task-sorting.spec.ts`).
- Built binary and placed it into `/home/bun/.local/bin/jokateko`. Task remains in 'in_review' awaiting human validation.

## Completion Summary
- **Completed At:** 2026-09-06T06:17:13Z

### What Was Done
Added multi-criteria task sorting comparator supporting priority tiers, target dates, recency (changed_at), creation timestamps, and titles. Configured column-level sort rules in config.toml with state-specific defaults (priority then target_at for active columns, priority then changed_at for backlog, and recently changed directly for done). Integrated per-column sort dropdowns into column header pills with independent state management in Preact signals. Updated MCP tools list_tasks and get_board_state to return consistently sorted tasks. Added full Go, Bun, and Playwright E2E test suites.

### Why / Rationale
Workflow stages have distinct prioritization dynamics: active tasks need deadline urgency visibility, backlog requires focus based on recency, and completed tasks benefit most from seeing recently finished work at the top without irrelevant priority ordering. Providing per-column controls gives users granular control over their board view.
