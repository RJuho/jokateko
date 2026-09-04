+++
title = 'Implement MCP Entity Deletion Tools with Safeguards'
status = 'done'
priority = 'high'
tags = ['api', 'backend']
summary = 'Implemented MCP entity deletion tools and aligned REST DELETE endpoints with dependency and assignment safeguards.'
+++

# Implement MCP Entity Deletion Tools with Safeguards

Implement entity deletion tools in MCP along with safety guards preventing accidental cascade or dangling references unless explicitly forced.

## Acceptance Criteria
- [x] Implement `delete_task` MCP tool with optional `force` boolean flag (rejects deletion if other tasks depend on this task unless `force: true`)
- [x] Implement `delete_milestone` MCP tool with optional `force` boolean flag (rejects deletion if tasks are assigned to this milestone unless `force: true`)
- [x] Implement `delete_strategy` MCP tool
- [x] Implement `delete_glossary_term` MCP tool
- [x] Align REST endpoints `DELETE /api/tasks/{id}` and `DELETE /api/milestones/{id}` to support `?force=true` and enforce the same dependency/assignment guards
- [x] Add unit tests for all deletion tools verifying safety guards and force overrides

## Completion Summary
- **Completed At:** 2026-09-04T08:09:02Z

### What Was Done
Added delete_task, delete_milestone, delete_strategy, and delete_glossary_term MCP tools. Implemented safety guards preventing deletion of tasks with downstream dependencies and milestones with assigned tasks unless force is true. Aligned REST DELETE endpoints for tasks and milestones with the same force query parameter and conflict safeguards, with unit tests covering all paths.

### Why / Rationale
Gives AI agents and REST clients safe and explicit deletion capabilities while protecting graph and entity integrity by default.
