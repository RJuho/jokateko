# CLI Commands Documentation

This document outlines the command-line interface for the application. The tool is designed to be run as a single static binary.

## Core Commands

### `[empty]` or `serve`
**Usage:** `jokateko` OR `jokateko serve`
**Description:** Starts the core application daemon. This initializes the in-memory SQLite database, starts the `fsnotify` file watcher, launches the local Web UI server (e.g., on port 8080), and opens the standard Model Context Protocol (MCP) server for AI agents to connect to. 

### `parse` (or `lint`)
**Usage:** `jokateko parse`
**Description:** Performs a strict dry-run validation of the entire project state. It reads all Markdown files (Tasks, Milestones, Strategies, Glossary) and the TOML configuration file, checking for broken YAML frontmatter, missing dependencies, invalid tags, and schema violations. Exits with code `0` if successful, or `1` with detailed error logs if formatting issues are found.

### `build`
**Usage:** `jokateko build`
**Description:** Generates a 100% self-contained, single-file HTML export of the current project state (CSS, JS, and all Tasks, Milestones, Strategies, and Glossary inlined into one `.html` file). This is ideal for publishing an offline snapshot to static hosting or emailing/sharing directly as a standalone file. *(Note: This does not compile the Go binary itself; it exports the user's project data and UI into a single offline HTML file).*

### `version`
**Usage:** `jokateko version`
**Description:** Outputs the current binary version, build date, and architecture (e.g., `v1.0.0-linux-amd64`).

### `help`
**Usage:** `jokateko help`
**Description:** Displays the global help menu, available commands, and flag descriptions.

---

## Suggested Additional Commands

### `init`
**Usage:** `jokateko init [-dir <path>] [-name <project-name>] [--replace]`
**Description:** Scaffolds a new project in the current directory. It generates the base folder structure (`.jokateko/tasks/`, `.jokateko/milestones/`, `.jokateko/strategies/`, `.jokateko/glossary/`), creates the default starter files, and writes a clean, self-documenting TOML configuration file (`.jokateko/config.toml`) with project settings active and all other sections commented out to inherit compiled-in defaults.
- `--replace`: Overwrite an existing `.jokateko/config.toml` with the fresh default commented template.

### `mcp`
**Usage:** `jokateko mcp`
**Description:** Start *only* as an MCP `stdio` server without attempting to bind to the web UI port. If your primary `serve` daemon is already running, this command acts as a lightweight proxy, routing the AI agent's `stdio` traffic to the running daemon's HTTP port to prevent database and port lock conflicts. If the daemon is not running, it runs standalone in stdio mode with an internal in-memory SQLite store.