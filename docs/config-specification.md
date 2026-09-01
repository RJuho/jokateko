# Jokateko Configuration Specification

This document details the configuration system for Jokateko, utilizing TOML via `github.com/pelletier/go-toml/v2`.

---

## 1. Overview & Principles

- **Format:** TOML (Tom's Obvious Minimal Language) v1.0.
- **Engine:** `github.com/pelletier/go-toml/v2` (pure Go, zero CGO, zero external binary requirement).
- **Default Location:** `.jokateko/config.toml` in the repository root.
- **Graceful Fallback:** If `.jokateko/config.toml` is absent, Jokateko uses compiled-in default configuration values without failing.
- **Validation:** Executed automatically during `jokateko serve`, `jokateko parse`, and `jokateko build`.

---

## 2. Configuration Schema & Example

Below is the complete reference `.jokateko/config.toml` file with all available sections, defaults, and explanatory comments:

```toml
# ==============================================================================
# Jokateko Project Configuration
# ==============================================================================

version = "0"

[project]
name = "My Project"
description = "Markdown-driven task management and Kanban"

# ------------------------------------------------------------------------------
# Storage Paths (relative to repository root or absolute)
# ------------------------------------------------------------------------------
[paths]
# Directory where task tickets (*.md) are stored
tasks = ".jokateko/tasks"

# Directory where milestone files (*.md) are stored
milestones = ".jokateko/milestones"

# Directory where architectural guidelines and rules are stored
strategies = ".jokateko/strategies"

# Directory where project glossary markdown files (*.md) are stored
glossary = ".jokateko/glossary"

# Default output directory for the 'jokateko build' command
export = "dist-kanban"

# ------------------------------------------------------------------------------
# Web UI & HTTP Server Settings (used by 'jokateko serve')
# ------------------------------------------------------------------------------
[server]
# Network interface to bind (127.0.0.1 for local only, 0.0.0.0 for container/network)
host = "127.0.0.1"

# HTTP port to listen on
port = 8080

# [TBD: Low Priority] Automatically open the web browser when 'serve' starts
open_browser = false

# ------------------------------------------------------------------------------
# Server Security: CORS & CSP (Content-Security-Policy)
# Note: These security headers only apply when running the local HTTP server ('serve').
# ------------------------------------------------------------------------------
[server.security]
# Enable Cross-Origin Resource Sharing (useful if running external dev servers or tools)
cors_enabled = false
cors_allowed_origins = ["http://localhost:3000", "http://127.0.0.1:3000"]

[server.security.csp]
# Enable Content-Security-Policy header injection
enabled = true

# Whitelist allowed sources for different resource types
# Users can customize these if loading external assets, fonts, or connecting to external APIs
default_src = ["'self'"]
script_src = ["'self'"]
style_src = ["'self'", "'unsafe-inline'"]
img_src = ["'self'", "data:"]
connect_src = ["'self'"]
font_src = ["'self'"]

# ------------------------------------------------------------------------------
# Kanban Board Workflow Columns
# Tasks are mapped to these columns via their frontmatter 'status' field.
# ------------------------------------------------------------------------------
[[board.columns]]
id = "backlog"
name = "Backlog"
color = "#94a3b8" # Slate

[[board.columns]]
id = "ready"
name = "Ready"
color = "#60a5fa" # Blue

[[board.columns]]
id = "in_progress"
name = "In Progress"
color = "#f59e0b" # Amber

[[board.columns]]
id = "in_review"
name = "In Review"
color = "#a855f7" # Purple

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981" # Emerald

# ------------------------------------------------------------------------------
# Controlled Tag Vocabulary
# Defines the allowed tags for tasks, milestones, and strategies.
# Prevents AI agents from creating ad-hoc or duplicate categorization tags.
# ------------------------------------------------------------------------------
[tags]
# Whitelisted list of tags allowed in the project
allowed = [
  "backend",
  "frontend",
  "database",
  "security",
  "ui",
  "auth",
  "api",
  "docs",
  "testing",
  "release",
  "infra"
]

# When true, task creation, updates, and 'jokateko parse' strictly reject unlisted tags
enforce_allowed = true

# ------------------------------------------------------------------------------
# Model Context Protocol (MCP) Configuration for AI Agents
# ------------------------------------------------------------------------------
[mcp]
# Enable built-in MCP server support
enabled = true

# Timeout in seconds for MCP tool executions
timeout_seconds = 30

# Expose mutation tools (create/update tasks) to AI agents
allow_mutations = true
```

---

## 3. Glossary Directory Format (`.jokateko/glossary/*.md`)

To completely prevent Git merge conflicts in team and multi-agent environments, glossary terms are stored as individual Markdown files inside the directory specified by `paths.glossary` (default `.jokateko/glossary/`).

Each term file contains YAML frontmatter (`title`, `tags`, `summary`) and a markdown body providing context:

```markdown
---
title: "Tasks-as-Code"
tags: ["methodology", "core"]
summary: "The methodology where actionable tasks and project specs are stored as version-controlled Markdown files in the repository, making them the absolute source of truth."
---

# Tasks-as-Code

The Tasks-as-Code methodology treats tasks, tickets, and specifications as version-controlled files in Git alongside application source code.
```

---

## 4. Go Configuration Structs

```go
package config

import "time"

type Config struct {
	Version string        `toml:"version"`
	Project ProjectConfig `toml:"project"`
	Paths   PathsConfig   `toml:"paths"`
	Server  ServerConfig  `toml:"server"`
	Board   BoardConfig   `toml:"board"`
	Tags    TagsConfig    `toml:"tags"`
	MCP     MCPConfig     `toml:"mcp"`
}

type ProjectConfig struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}

type PathsConfig struct {
	Tasks      string `toml:"tasks"`
	Milestones string `toml:"milestones"`
	Strategies string `toml:"strategies"`
	Glossary   string `toml:"glossary"`
	Export     string `toml:"export"`
}

type ServerConfig struct {
	Host        string               `toml:"host"`
	Port        int                  `toml:"port"`
	OpenBrowser bool                 `toml:"open_browser"` // TBD: Low priority
	Security    ServerSecurityConfig `toml:"security"`
}

type ServerSecurityConfig struct {
	CORSEnabled        bool      `toml:"cors_enabled"`
	CORSAllowedOrigins []string  `toml:"cors_allowed_origins"`
	CSP                CSPConfig `toml:"csp"`
}

type CSPConfig struct {
	Enabled    bool     `toml:"enabled"`
	DefaultSrc []string `toml:"default_src"`
	ScriptSrc  []string `toml:"script_src"`
	StyleSrc   []string `toml:"style_src"`
	ImgSrc     []string `toml:"img_src"`
	ConnectSrc []string `toml:"connect_src"`
	FontSrc    []string `toml:"font_src"`
}

type BoardConfig struct {
	Columns []ColumnConfig `toml:"columns"`
}

type ColumnConfig struct {
	ID    string `toml:"id"`
	Name  string `toml:"name"`
	Color string `toml:"color"`
}

type TagsConfig struct {
	Allowed        []string `toml:"allowed"`
	EnforceAllowed bool     `toml:"enforce_allowed"`
}

type MCPConfig struct {
	Enabled        bool          `toml:"enabled"`
	TimeoutSeconds time.Duration `toml:"timeout_seconds"`
	AllowMutations bool          `toml:"allow_mutations"`
}

// Glossary model (parsed from .jokateko/glossary/*.md files)
type GlossaryTerm struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
	Body    string   `json:"body"`
}
```

---

## 5. Default Configuration & Overrides

### 5.1 Built-in Defaults
When no `config.toml` exists in the repository, `config.Default()` returns:
- `project.name` = Directory basename
- `paths.tasks` = `.jokateko/tasks`
- `paths.milestones` = `.jokateko/milestones`
- `paths.strategies` = `.jokateko/strategies`
- `paths.glossary` = `.jokateko/glossary`
- `paths.export` = `dist-kanban`
- `server.host` = `127.0.0.1`
- `server.port` = `8080`
- `server.open_browser` = `false`
- `server.security.cors_enabled` = `false`
- `server.security.csp.enabled` = `true`
- `board.columns` = `[backlog, ready, in_progress, in_review, done]`
- `tags.allowed` = `["backend", "frontend", "database", "security", "ui", "auth", "api", "docs", "testing", "release", "infra"]`
- `tags.enforce_allowed` = `true`
- `mcp.enabled` = `true`
- `mcp.allow_mutations` = `true`

### 5.2 Environment Variable Overrides
Jokateko supports runtime environment variables to override critical settings without editing `config.toml`:
- `JOKATEKO_CONFIG`: Path to a custom config file (default: `.jokateko/config.toml`).
- `JOKATEKO_HOST`: Host interface to bind (e.g. `0.0.0.0` inside containers).
- `JOKATEKO_PORT`: Port number to bind (e.g. `8080`).

---

## 6. Configuration Validation Rules

During `jokateko parse`, the configuration is validated against the following invariants:

1. **Version Format (`CFG-000`):** `version` must match a supported schema version (initially `"0"`).
2. **Port Range (`CFG-004`):** `server.port` must be between `1024` and `65535`.
3. **Column Uniqueness (`CFG-003`):** Each column in `board.columns` must have a non-empty, unique `id`.
4. **Column Minimum (`CFG-002`):** At least 2 columns must be defined (e.g. Backlog and Done).
5. **Path Sanitization (`CFG-005`):** Paths must not resolve outside the workspace root (no `../../` path traversal).
6. **Color Format:** If `color` is provided, it must be a valid hex code (`#rrggbb` or `#rgb`).
7. **CSP Directives:** If `csp.enabled` is true, source lists must contain valid CSP tokens (e.g. `'self'`, `data:`, or valid host origins).
8. **Tags Allowed List (`TAG-001`, `TAG-002`):** If `tags.enforce_allowed` is true, `tags.allowed` must contain at least 1 tag, and tag names must follow kebab-case (`^[a-z0-9]+(-[a-z0-9]+)*$`).
