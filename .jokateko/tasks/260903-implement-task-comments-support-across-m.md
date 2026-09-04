+++
title = 'Implement Task Comments Support Across MCP, REST, and Web UI'
status = 'backlog'
priority = 'medium'
tags = ['backend', 'frontend', 'api', 'feature']
summary = 'Add add_task_comment MCP tool, REST endpoint, and web UI comment section in TaskDetailModal to append timestamped notes under ## Comments.'
+++

# Implement Task Comments Support Across MCP, REST, and Web UI

Allow agents and humans to append progress notes and work logs to a task without overwriting existing description or acceptance criteria.

## Acceptance Criteria
- [ ] Implement `add_task_comment` MCP tool taking `id`, optional `author` (defaulting to "Agent"), and required `comment`
- [ ] Appends formatted entries `### [YYYY-MM-DD HH:MM UTC] - Author` and comment body under a `## Comments` section in the task markdown
- [ ] Implement `POST /api/tasks/{id}/comments` REST endpoint accepting `{ "author": string, "comment": string }`
- [ ] Broadcast `task.commented` SSE event when a comment is added and refresh active task in store
- [ ] Update `TaskDetailModal.tsx` in Web UI with a comment entry box and submit button allowing human users to post comments
- [ ] Add unit tests for `add_task_comment` and REST endpoint verifying correct markdown formatting and error handling
