+++
title = 'Implement Configurable AI Agent Guidance and Commented Init Config Template'
status = 'done'
priority = 'high'
tags = ['backend', 'feature']
summary = 'Configurable MCP server AI guidance via [mcp.instructions] and commented-out config template generator for init derived directly from default.toml.'
+++

# Implement Configurable AI Agent Guidance and Commented Init Config Template

Enable project owners to customize the AI agent instructions exposed by the MCP server, and generate a clean, self-documenting `.jokateko/config.toml` on `init` where only the project name and description are active while all other sections are commented out.

## Acceptance Criteria
- [x] Add `instructions` field to `MCPConfig` struct in `internal/config/config.go` and update `internal/config/default.toml` with comprehensive default guidance
- [x] Wire `cfg.MCP.Instructions` into `internal/mcp/mcp.go`, replacing the hardcoded instructions when provided (falling back to built-in default if empty)
- [x] Update MCP server default instructions to include explicit guidelines to use MCP tools rather than direct file edits for all `.jokateko/` modifications
- [x] Implement template generator or transformation mechanism in `internal/config` so `init` creates `.jokateko/config.toml` directly from `default.toml` as the single source of truth
- [x] Update `cmd/jokateko/init.go` so generated `config.toml` has only `version = "0"` and `[project]` (name and description) uncommented, with all other sections commented out
- [x] Add unit tests verifying:
  - Custom `[mcp.instructions]` in TOML replaces the default instructions in the MCP server options
  - Omitted `[mcp.instructions]` uses the default instructions
  - `init` command writes a valid commented config that parses correctly and inherits all compiled-in defaults

## Completion Summary
- **Completed At:** 2026-09-04T07:47:24Z

### What Was Done
Added Instructions field to MCPConfig and rawMCPConfig, updated default.toml with comprehensive system guidance, wired custom instructions into MCP server options with fallback to DefaultInstructions, added GenerateCommentedConfig to dynamically create a clean commented template from embedded default.toml, updated cmdInit to use the template, and added unit tests.

### Why / Rationale
Gives project owners full control over AI agent system guidance through config.toml, enforces that AI agents use MCP mutation tools rather than raw filesystem edits, and ensures jokateko init creates a self-documenting configuration file while inheriting all compiled-in defaults.
