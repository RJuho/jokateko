+++
title = 'Multi-Criteria Task Sorting in Web UI and MCP'
status = 'backlog'
priority = 'medium'
tags = ['backend', 'frontend', 'feature']
summary = 'Implement configurable multi-criteria task sorting per column (Priority tier with state-dependent secondary: changed_at desc for backlog/done, soonest target_at asc for active states) with Web UI Sort dropdown and MCP list alignment.'
dependencies = ['260904-task-and-milestone-timestamps-createdat-']
created_at = '2026-09-05T17:02:29Z'
changed_at = '2026-09-05T17:02:29Z'
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
- [ ] Implement multi-criteria sorting comparator supporting priority tiers, target_at (ascending), and changed_at (descending)
- [ ] Support optional per-column sort configurations in `[[board.columns]]` with fallback defaults
- [ ] Add Sort dropdown button to Web UI `FilterBar` with interactive sort mode selection
- [ ] Update Preact store `columnTasks` computed signal to reactively sort tasks per column
- [ ] Align MCP `list_tasks` and `get_board_state` output ordering with column default sorting
- [ ] Add comprehensive unit tests in Go and Bun for sorting determinism and edge cases (missing target_at, ties)
- [ ] Add Playwright E2E tests validating Sort By dropdown interactions and board card ordering
