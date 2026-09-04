+++
title = 'Update Agent Mutation Guidelines and MCP Specifications'
status = 'done'
priority = 'medium'
tags = ['docs']
summary = 'Update MCP server instructions, AGENTS.md, and docs/mcp-specification.md to enforce MCP mutations over raw file edits and document all new tools.'
dependencies = ['260903-implement-explicit-task-dependency-manag', '260903-implement-mcp-entity-deletion-tools-with', '260904-implement-task-notes-and-configurable-st']
+++

# Update Agent Mutation Guidelines and MCP Specifications

Ensure all AI agents understand that .jokateko/ mutations must be performed via MCP tools rather than direct file edits, and document all newly added MCP tools in the specification.

## Acceptance Criteria
- [x] Update `internal/mcp/mcp.go` server Instructions string to instruct agents to use MCP mutation tools instead of raw filesystem writes
- [x] Update `AGENTS.md` with explicit guidelines warning agents against direct file modifications to `.jokateko/`
- [x] Document new tools (`delete_task`, `delete_milestone`, `delete_strategy`, `delete_glossary_term`, `add_task_dependency`, `remove_task_dependency`, `add_task_note`) in `docs/mcp-specification.md`
- [x] Verify test suite passes with `go test ./...`

## Completion Summary
- **Completed At:** 2026-09-04T12:00:55Z

### What Was Done
Updated DefaultInstructions in mcp.go and default.toml with explicit rule 6; added Section 5 and dynamic tool discovery in AGENTS.md; documented delete tools, dependency management tools, add_task_note, strategy/glossary mutation tools, and Section 1.1 safe mutation rules in docs/mcp-specification.md.

### Why / Rationale
Ensures AI agents avoid direct edits to .jokateko/ markdown files which bypass schema validation, cache suppression, DAG cycle checks, and real-time SSE broadcasts.
