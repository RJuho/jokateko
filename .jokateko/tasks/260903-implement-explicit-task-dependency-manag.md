+++
title = 'Implement Explicit Task Dependency Management Tools'
status = 'backlog'
priority = 'high'
tags = ['backend', 'api']
summary = 'Add add_task_dependency and remove_task_dependency tools to MCP with cycle detection and validation.'
+++

# Implement Explicit Task Dependency Management Tools

Provide dedicated MCP tools and REST endpoints to manage task dependencies without needing full task rewrites, enforcing cycle detection and graph integrity.

## Acceptance Criteria
- [ ] Implement `add_task_dependency` MCP tool taking `id` and `dependency_id`
- [ ] Validate target and dependency task existence and reject self-dependencies (`id == dependency_id`)
- [ ] Run DAG cycle validation (`validator.DetectCycles`) before committing edge addition, rejecting any cycles with descriptive errors
- [ ] Implement `remove_task_dependency` MCP tool taking `id` and `dependency_id`
- [ ] Add matching REST endpoints `POST /api/tasks/{id}/dependencies` and `DELETE /api/tasks/{id}/dependencies/{depId}`
- [ ] Add unit tests for adding/removing dependencies, duplicate edge handling, and cycle rejection
