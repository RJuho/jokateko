+++
title = 'Implement Configurable AI Agent Guidance and Commented Init Config Template'
status = 'backlog'
priority = 'high'
tags = ['backend', 'feature']
summary = 'Make MCP server instructions configurable via [mcp.instructions] multi-line TOML, and generate fully commented-out config on init derived directly from default.toml.'
+++

# Implement Configurable AI Agent Guidance and Commented Init Config Template

Enable project owners to customize the AI agent instructions exposed by the MCP server, and generate a clean, self-documenting `.jokateko/config.toml` on `init` where only the project name and description are active while all other sections are commented out.

## Acceptance Criteria
- [ ] Add `instructions` field to `MCPConfig` struct in `internal/config/config.go` and update `internal/config/default.toml` with comprehensive default guidance
- [ ] Wire `cfg.MCP.Instructions` into `internal/mcp/mcp.go`, replacing the hardcoded instructions when provided (falling back to built-in default if empty)
- [ ] Update MCP server default instructions to include explicit guidelines to use MCP tools rather than direct file edits for all `.jokateko/` modifications
- [ ] Implement template generator or transformation mechanism in `internal/config` so `init` creates `.jokateko/config.toml` directly from `default.toml` as the single source of truth
- [ ] Update `cmd/jokateko/init.go` so generated `config.toml` has only `version = "0"` and `[project]` (name and description) uncommented, with all other sections commented out
- [ ] Add unit tests verifying:
  - Custom `[mcp.instructions]` in TOML replaces the default instructions in the MCP server options
  - Omitted `[mcp.instructions]` uses the default instructions
  - `init` command writes a valid commented config that parses correctly and inherits all compiled-in defaults
