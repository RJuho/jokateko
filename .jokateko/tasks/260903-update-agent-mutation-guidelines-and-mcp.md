+++
title = 'Update Agent Mutation Guidelines and MCP Specifications'
status = 'backlog'
priority = 'medium'
tags = ['docs']
summary = 'Update MCP server instructions, AGENTS.md, and docs/mcp-specification.md to document new tools and enforce safe MCP mutations over raw file edits.'
dependencies = ['260903-implement-explicit-task-dependency-manag', '260903-implement-mcp-entity-deletion-tools-with', '260904-implement-task-notes-and-configurable-st']
+++

# Update Agent Mutation Guidelines and MCP Specifications

Ensure all AI agents understand that .jokateko/ mutations must be performed via MCP tools rather than direct file edits, and document all newly added MCP tools in the specification.

## Acceptance Criteria
- [ ] Update `internal/mcp/mcp.go` server Instructions string to instruct agents to use MCP mutation tools instead of raw filesystem writes
- [ ] Update `AGENTS.md` with explicit guidelines warning agents against direct file modifications to `.jokateko/`
- [ ] Document new tools (`delete_task`, `delete_milestone`, `delete_strategy`, `delete_glossary_term`, `add_task_dependency`, `remove_task_dependency`, `add_task_note`) in `docs/mcp-specification.md`
- [ ] Verify test suite passes with `go test ./...`
