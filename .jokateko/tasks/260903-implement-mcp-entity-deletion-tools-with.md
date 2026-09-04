+++
title = 'Implement MCP Entity Deletion Tools with Safeguards'
status = 'backlog'
priority = 'high'
tags = ['backend', 'api']
summary = 'Add delete_task, delete_milestone, delete_strategy, and delete_glossary_term tools to MCP with dependency and reference safeguards.'
+++

# Implement MCP Entity Deletion Tools with Safeguards

Implement entity deletion tools in MCP along with safety guards preventing accidental cascade or dangling references unless explicitly forced.

## Acceptance Criteria
- [ ] Implement `delete_task` MCP tool with optional `force` boolean flag (rejects deletion if other tasks depend on this task unless `force: true`)
- [ ] Implement `delete_milestone` MCP tool with optional `force` boolean flag (rejects deletion if tasks are assigned to this milestone unless `force: true`)
- [ ] Implement `delete_strategy` MCP tool
- [ ] Implement `delete_glossary_term` MCP tool
- [ ] Align REST endpoints `DELETE /api/tasks/{id}` and `DELETE /api/milestones/{id}` to support `?force=true` and enforce the same dependency/assignment guards
- [ ] Add unit tests for all deletion tools verifying safety guards and force overrides
