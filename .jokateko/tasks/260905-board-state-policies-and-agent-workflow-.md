+++
title = 'Board State Policies and Agent Workflow Configuration'
status = 'backlog'
priority = 'medium'
tags = ['backend', 'frontend', 'feature']
summary = "Configure board column creation permissions (restricting '+' task button to backlog by default) and agent workflow state ownership (handled_by, instructions) served via MCP system instructions, get_board_state, and Web UI indicators."
created_at = '2026-09-05T17:02:26Z'
changed_at = '2026-09-05T17:02:26Z'
+++

# Board State Policies and Agent Workflow Configuration

Establish configurable board column creation restrictions and agent workflow role guidance in Jokateko configuration, exposing them through the MCP server and Web UI.

## Background & Rationale
In multi-agent and human-agent collaborative environments, tasks should follow disciplined lifecycles. New work typically enters the workflow through the Backlog rather than intermediate or review stages. Furthermore, specific workflow states (such as implementation, code review, or QA) are often designated for specific agent personas, models, or human roles. Providing explicit configuration in `.jokateko/config.toml` allows the MCP server to inform AI agents of role boundaries and ensures the Web UI reflects allowed creation states.

## Requirements & Scope
1. **Board Configuration Extensions (`config.toml`)**:
   - `creatable_states`: Array of column IDs where new tasks can be created (default `["backlog"]`).
   - `default_create_state`: Default column ID for newly created tasks (default `"backlog"`).
   - `handled_by`: Optional string per column definition specifying the assigned agent persona, model, or team role (e.g. `"agent:coder"`, `"agent:reviewer"`, `"human"`).
   - `instructions`: Optional workflow guidance string per column detailing state exit/entry criteria.
2. **Web UI Column Header & Modal Updates**:
   - In `Column.tsx`: Only display the "+" add-task button for columns listed in `creatable_states` (restricting by default to Backlog).
   - Display a subtle role badge or icon on column headers when `handled_by` is configured.
   - In `CreateTaskModal.tsx`: Set default status to `default_create_state` (`"backlog"`).
3. **MCP Server Integration**:
   - In `internal/mcp/mcp.go`: Dynamically format and append workflow role guidance and column state ownership into the MCP server instructions provided to AI agents.
   - In `get_board_state`: Include `handled_by` and `instructions` in `BoardColumnSummary`.
   - In `create_task`: Default status to `default_create_state` if status is not explicitly provided.

## Acceptance Criteria
- [ ] Add `creatable_states` and `default_create_state` to `BoardConfig` in Go and Valibot schemas
- [ ] Add `handled_by` and `instructions` to `ColumnConfig` in Go and Valibot schemas
- [ ] Update default configuration and documentation to set `creatable_states = ["backlog"]`
- [ ] Restrict "+" button in Web UI Kanban column headers to columns in `creatable_states`
- [ ] Update `CreateTaskModal` to default initial status to `default_create_state`
- [ ] Render role badge or tooltip in Web UI column headers when `handled_by` is configured
- [ ] Expose column role metadata in MCP `get_board_state` tool output
- [ ] Dynamically inject column role and workflow instructions into MCP server system prompt
- [ ] Add unit and integration tests for config loading, MCP instructions, and UI column permissions
