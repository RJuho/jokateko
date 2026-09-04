+++
title = 'Implement Explicit Task Dependency Management Tools'
status = 'done'
priority = 'high'
tags = ['api', 'backend']
summary = 'Implemented explicit task dependency management tools (add_task_dependency, remove_task_dependency) with cycle detection and matching REST endpoints.'
+++

# Implement Explicit Task Dependency Management Tools

Provide dedicated MCP tools and REST endpoints to manage task dependencies without needing full task rewrites, enforcing cycle detection and graph integrity.

## Acceptance Criteria
- [x] Implement `add_task_dependency` MCP tool taking `id` and `dependency_id`
- [x] Validate target and dependency task existence and reject self-dependencies (`id == dependency_id`)
- [x] Run DAG cycle validation (`validator.DetectCycles`) before committing edge addition, rejecting any cycles with descriptive errors
- [x] Implement `remove_task_dependency` MCP tool taking `id` and `dependency_id`
- [x] Add matching REST endpoints `POST /api/tasks/{id}/dependencies` and `DELETE /api/tasks/{id}/dependencies/{depId}`
- [x] Add unit tests for adding/removing dependencies, duplicate edge handling, and cycle rejection

## Completion Summary
- **Completed At:** 2026-09-04T11:23:04Z

### What Was Done
Added add_task_dependency and remove_task_dependency MCP tools with cycle detection (validator.DetectCycles), self-dependency validation, and duplicate edge handling. Added REST endpoints POST /api/tasks/{id}/dependencies and DELETE /api/tasks/{id}/dependencies/{depId}. Added automated unit tests for MCP tools and REST endpoints.

### Why / Rationale
Enables explicit, safe, and granular dependency management without requiring full task markdown rewrites, protecting DAG integrity.
