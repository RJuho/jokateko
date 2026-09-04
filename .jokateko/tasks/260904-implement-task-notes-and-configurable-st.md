+++
title = 'Implement Task Notes and Configurable State Editing'
status = 'done'
priority = 'medium'
tags = ['api', 'backend', 'feature', 'frontend']
summary = 'Add add_task_note MCP tool, REST notes endpoint, configurable editable states in config.toml, and Web UI note/spec editor support.'
+++

# Implement Task Notes and Configurable State Editing

Provide task note appending across MCP, REST, and Web UI, and enforce configurable workflow states where full task body editing is permitted (defaulting to backlog).

## Acceptance Criteria
- [x] Add `editable_states` configuration under `[board]` in `config.toml` (defaulting to `["backlog"]`) with helper `Config.IsTaskEditable(status)`
- [x] Guard task body editing in `update_task_content` and `PUT /api/tasks/{id}`: allow full body editing only when task status is in `editable_states`, rejecting with a descriptive error otherwise
- [x] Implement `add_task_note` MCP tool taking `id` and `note` (markdown string, no author), appending timestamped notes under `## Notes`
- [x] Implement matching REST endpoint `POST /api/tasks/{id}/notes` accepting `{ "note": string }`
- [x] Ensure any checkboxes `- [ ]` added inside `## Notes` are tracked as required acceptance criteria and block `complete_task` until resolved
- [x] Update `TaskDetailModal.tsx` in Web UI: show full body editor when task is in an editable state, and show Add Note section when task is in non-editable states
- [x] Add unit tests for configurable editable state guards, `add_task_note`, REST endpoint, and notes checkbox enforcement

## Completion Summary
- **Completed At:** 2026-09-04T11:55:18Z

### What Was Done
Added editable_states to config.toml and BoardConfig; guarded task body editing in MCP and REST to only permit modifications in editable states while allowing checkbox toggling anywhere; implemented add_task_note MCP tool and POST /api/tasks/{id}/notes REST endpoint; ensured notes checkboxes are tracked acceptance criteria; updated TaskDetailModal with specification editor and Add Note box, and TaskEditModal with locked state; added comprehensive unit tests.

### Why / Rationale
Allows AI agents and human developers to safely append progress notes and follow-up criteria across workflow states while protecting the specification from unintended overwrites once past backlog.
