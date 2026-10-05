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
**Usage:** `jokateko build [-dir <path>] [-out <file>] [--mermaidjs=cdn|bundled|none]`
**Description:** Generates a single-file HTML export of the current project state (CSS, JS, and all Tasks, Milestones, Strategies, and Glossary inlined into one `.html` file). This is ideal for publishing a snapshot to static hosting or emailing/sharing directly as a standalone file. *(Note: This does not compile the Go binary itself; it exports the user's project data and UI into a single HTML file).*
- `--mermaidjs` selects how Mermaid diagrams are rendered in the export:
  - `cdn` *(default)*: loads the exact Mermaid version pinned in `web/bun.lock` from jsDelivr when a diagram is shown, verified with Subresource Integrity (SRI). The file stays small (~0.5 MB), but diagrams need network access.
  - `bundled`: inlines the Mermaid runtime (~5.5 MB) as an inert block, executed only when a diagram is shown. The export is 100% self-contained and works fully offline.
  - `none`: no Mermaid runtime; diagrams are shown as their source code blocks.

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

---

## Development Make Targets

### `make cover`
**Usage:** `make cover`
**Description:** Runs the Go tests under `./cmd/...` and `./internal/...` with CGO disabled, writes a coverage profile to `coverage.out` (git-ignored), and prints the total statement coverage. Like `make test`, it first rebuilds the Web UI bundle when its sources changed and generates the license report if it is missing, because the code under test embeds both. Use `go tool cover -func=coverage.out` for per-function figures or `go tool cover -html=coverage.out` to browse uncovered lines.
