# Jokateko Module Structure

This document outlines the Go backend module architecture, internal packages, domain boundaries, data flow pipelines, and separation of concerns.

---

## 1. Go Module Definition

- **Module Path:** `github.com/RJuho/jokateko` (or `jokateko`)
- **Go Version:** `1.23+` (compatible with `go1.27+` in devcontainer)
- **Strict Constraint:** Zero CGO dependencies (`CGO_ENABLED=0`).

### Allowed Core Dependencies

| Dependency | Purpose | Rationale |
|---|---|---|
| `modernc.org/sqlite` | Pure Go SQLite engine | Zero CGO, fast in-memory execution, cross-platform |
| `github.com/pelletier/go-toml/v2` | TOML parser & encoder | Pure Go, spec-compliant, zero external runtime dependency |
| `github.com/fsnotify/fsnotify` | Filesystem change events | Cross-platform event-driven file monitoring |
| `github.com/yuin/goldmark` | Markdown parsing | Fast, extensible, standard-compliant CommonMark |
| `github.com/modelcontextprotocol/go-sdk` | Model Context Protocol | Official MCP SDK for AI agent tools & resources |

---

## 2. Package Organization

The codebase follows standard Go project layout conventions (`cmd/`, `internal/`, `web/`):

```text
jokateko/
├── cmd/
│   └── jokateko/               # Main executable entrypoint
│       ├── main.go             # Entrypoint, root flag router
│       ├── serve.go            # 'serve' subcommand
│       ├── parse.go            # 'parse' / 'lint' subcommand
│       ├── build.go            # 'build' static export subcommand
│       ├── init.go             # 'init' scaffolding subcommand
│       ├── mcp.go              # 'mcp' stdio / proxy subcommand
│       └── version.go          # 'version' subcommand
├── internal/
│   ├── config/                 # TOML configuration loader & schema
│   ├── model/                  # Core domain types & enums
│   ├── parser/                 # Frontmatter (YAML) & Markdown parsing (goldmark)
│   ├── store/                  # In-memory SQLite, sqlc schema & queries
│   ├── watcher/                # fsnotify unidirectional file observer & debouncer
│   ├── writer/                 # Safe atomic file writes (temp file + rename)
│   ├── validator/              # Dry-run linting & dependency graph validation
│   ├── service/                # Shared REST/MCP mutation workflows (validation, IDs, persistence, events)
│   ├── server/                 # HTTP server, REST endpoints, SSE hub, static embed
│   ├── mcp/                    # Official MCP Go SDK server & tool definitions
│   ├── proxy/                  # MCP stdio proxy to running daemon HTTP server
│   └── exporter/               # Static site generator for 'build' command
├── web/                        # Preact frontend source code (see web-ui-architecture.md)
│   ├── src/
│   ├── dist/                   # Compiled static bundle (embedded into Go)
│   ├── embed.go                # //go:embed dist/* directive
│   └── package.json
├── docs/                       # Project architecture specifications
├── AGENTS.md                   # Operational guidelines for AI agents
├── COMMANDS.md                 # CLI usage documentation
└── go.mod
```

---

## 3. Package Responsibilities & Boundaries

### `cmd/jokateko/`
- **Role:** CLI dispatch and lifecycle orchestration.
- **Responsibilities:**
  - Parses subcommands and flags using standard Go `flag.FlagSet` (zero external CLI dependencies).
  - Handles OS signals (`SIGINT`, `SIGTERM`) for graceful shutdown.
  - Instantiates and wires together `internal/` services based on the invoked command.

### `internal/config/`
- **Role:** Configuration management.
- **Responsibilities:**
  - Loads `.jokateko/config.toml` using `pelletier/go-toml/v2`.
  - Applies sensible defaults (port `8080`, host `127.0.0.1`, default directories).
  - Validates configuration values (e.g. valid port ranges, column status names).
  - Exposes configuration structs to other packages.

### `internal/model/`
- **Role:** Domain models and business entities.
- **Responsibilities:**
  - Defines types for `Task`, `Milestone`, `Strategy`, `GlossaryTerm`, `Tag`, and `BoardState`.
  - Defines standard workflow status enums (e.g., `backlog`, `ready`, `in_progress`, `in_review`, `done`).
  - Defines filter criteria and sorting options.

### `internal/parser/`
- **Role:** Markdown and YAML frontmatter processing.
- **Responsibilities:**
  - Splits raw markdown files into YAML frontmatter and markdown body.
  - Decodes and validates frontmatter fields into domain models.
  - Uses `yuin/goldmark` for AST inspection and HTML rendering where necessary.
  - Reports line-precise parsing diagnostics.

### `internal/store/`
- **Role:** In-memory query indexing and relational aggregation.
- **Responsibilities:**
  - Manages a pure in-memory SQLite connection (`file::memory:?cache=shared`).
  - Executes embedded DDL migrations on startup to create tables: `tasks`, `milestones`, `strategies`, `glossary`, `tags`, `dependencies`.
  - Uses `sqlc`-generated queries for type-safe SQLite interaction.
  - Supports atomic batch refreshes and individual file upserts/deletions.

### `internal/watcher/`
- **Role:** Real-time unidirectional filesystem tracking.
- **Responsibilities:**
  - Wraps `fsnotify.Watcher` to monitor `.jokateko/` subdirectories (`tasks/`, `milestones/`, `strategies/`, `glossary/`) and the root configuration file (`config.toml`).
  - Implements a debounce queue (e.g. 50ms) to coalesce rapid write/flush operations.
  - Filters out editor artifacts (e.g. `.swp`, `~`, `.DS_Store`).
  - Triggers re-parsing in `internal/parser/` and upserts into `internal/store/`.

### `internal/writer/`
- **Role:** Safe, atomic file persistence.
- **Responsibilities:**
  - All file mutations initiated via the Web UI or MCP MUST pass through this package.
  - Employs atomic write pattern: write to a temporary file (`.filename.tmp`) in the target directory, `fsync`, and atomically rename over the destination.
  - Suppresses self-triggered watcher events or synchronizes them cleanly to prevent feedback loops.

### `internal/validator/`
- **Role:** Project-wide linting and graph integrity verification (`jokateko parse`).
- **Responsibilities:**
  - Scans all files and validates frontmatter schemas against strict rules.
  - Builds a Directed Acyclic Graph (DAG) of task dependencies and detects circular dependencies using Tarjan's or DFS cycle detection.
  - Checks for dangling references (tasks referencing nonexistent milestones or dependency slugs).
  - Returns human-readable, colored diagnostic errors with file and line references.

### `internal/server/`
- **Role:** Local HTTP service, REST API, and Server-Sent Events (SSE).
- **Responsibilities:**
  - Serves static Web UI assets from `web/dist` using `http.FS`.
  - Implements REST endpoints for UI queries and mutations:
    - `GET /api/board` -> Columns and tasks aggregated
    - `GET /api/tasks`, `GET /api/tasks/{id}`, `POST /api/tasks`, `PATCH /api/tasks/{id}`
    - `GET /api/milestones`, `POST /api/milestones`
    - `GET /api/strategies`, `GET /api/strategies/{id}`
    - `GET /api/glossary`
    - `GET /api/health` -> Health check endpoint for status and CLI proxy discovery.
    - `GET /api/events` -> SSE stream broadcasting entity change events (`task.created`, `task.updated`, `task.deleted`, `board.refreshed`).
  - Security: rejects cross-site state-changing requests (`http.CrossOriginProtection`; configured CORS origins stay trusted), rejects non-loopback `Host` headers on loopback connections (DNS rebinding), requires `application/json` for POST/PUT bodies, and caps bodies at 1 MiB.

### `internal/service/`
- **Role:** Shared mutation workflows for the REST API and MCP tools.
- **Responsibilities:**
  - Validates input, including entity IDs (lowercase slugs only, so they are always safe file names) and board columns.
  - Allocates IDs: explicit duplicates are rejected with a conflict, generated IDs get a `-2`, `-3`, ... suffix instead of overwriting.
  - Writes Markdown atomically via `internal/writer/`, re-indexes `internal/store/`, and broadcasts SSE change events.
  - Serializes mutations so concurrent REST and MCP read-modify-write cycles cannot lose updates.

### `internal/mcp/`
- **Role:** Model Context Protocol implementation for AI agents.
- **Responsibilities:**
  - Implements the official MCP Go SDK (`modelcontextprotocol/go-sdk`).
  - Exposes tools: `list_tasks`, `get_task`, `create_task`, `update_task_status`, `complete_task`, `update_task_content`, `list_milestones`, `create_milestone`, `update_milestone`, `list_strategies`, `get_strategy`, `lookup_glossary`, `search_tasks`, `search_milestones`, `search_strategies`, `search_glossary`, `search_all`, `list_tags`, `get_board_state`.
  - Exposes resources: `jokateko://board`, `jokateko://strategies/tier1`, `jokateko://glossary`.
  - Exposes prompt templates: `next_task`.
  - Queries `internal/store/` and executes mutations via `internal/service/`, so agent changes are broadcast to the Web UI over SSE.

### `internal/proxy/`
- **Role:** Stdio MCP proxy to running daemon.
- **Responsibilities:**
  - Used by `jokateko mcp` when `jokateko serve` is already running.
  - Sends a health probe to `http://127.0.0.1:<port>/api/health` using the configured server port, and requires the reported `workspace` to match.
  - If daemon responds, bridges incoming stdio JSON-RPC messages to the daemon's Streamable HTTP endpoint.
  - If daemon is NOT running, launches standalone in-memory store and MCP server directly within the process.

### `internal/exporter/`
- **Role:** Standalone static build generation (`jokateko build`).
- **Responsibilities:**
  - Gathers full state from `internal/store/` (tasks, milestones, strategies, glossary, config).
  - Serializes state into a compact JSON payload.
  - Inlines the compiled CSS into a `<style>` tag, inlines the bundled Preact JavaScript into a `<script>` tag, and injects the JSON snapshot into `<script id="jokateko-data" type="application/json">`.
  - Outputs a 100% self-contained, single `.html` file (e.g. `dist-kanban/index.html` or specified file) requiring zero external assets, runnable offline from any filesystem or browser.

---

## 4. End-to-End Data Flow Pipelines

### 4.1 Ingestion & File Watch Pipeline (Serve Mode)

```text
[User or Git edits Markdown]
        │
        ▼
[Filesystem Event] ──► [fsnotify.Watcher]
                             │ (debounce 50ms)
                             ▼
                    [internal/parser]
                    - Extract YAML Frontmatter
                    - Parse Markdown AST
                             │
                             ▼
                    [internal/store]
                    - Upsert in-memory SQLite tables
                             │
                             ▼
                    [internal/server (SSE Hub)]
                    - Broadcast event { "type": "task.updated", "id": "..." }
                             │
                             ▼
                    [Web Browser UI]
                    - Preact state updates dynamically
```

### 4.2 Web UI Mutation Pipeline

```text
[User drags Kanban Card in Browser]
        │
        ▼
[PATCH /api/tasks/260901-task-slug]
        │
        ▼
[internal/server Handler]
        │ (validate input)
        ▼
[internal/writer]
        │ (atomic write: .260901-task-slug.md.tmp -> rename)
        ▼
[Filesystem: .jokateko/tasks/260901-task-slug.md]
        │
        ▼
[fsnotify Ingestion Pipeline picks up change and syncs SSE]
```

### 4.3 MCP AI Agent Pipeline

```text
[AI Agent (Cursor / Claude / Antigravity)]
        │ (stdio JSON-RPC)
        ▼
[jokateko mcp]
        ├── If daemon running? ──► Proxy to [http://localhost:8080/api/mcp]
        └── If standalone?     ──► Internal MCP Server
                                          │
                                ┌─────────┴─────────┐
                                ▼                   ▼
                         Read Queries         Mutation Calls
                                │                   │
                         [internal/store]    [internal/writer]
                         (in-memory SQLite)  (atomic file write)
```

---

## 5. Concurrency & Synchronization Model

1. **Single Writer / Mutex for In-Memory SQLite:**
   - SQLite in `:memory:` mode is protected by a `sync.RWMutex` across all goroutines.
2. **Atomic File Replacement:**
   - No file is ever written partially. Every mutation writes to a temporary file in the same directory and uses `os.Rename` (atomic POSIX operation).
3. **Watcher Echo Prevention:**
   - The writer records recently touched file paths and timestamps in an internal short-lived LRU/cache. When `fsnotify` fires for a self-originated write, the watcher recognizes the hash/timestamp and suppresses duplicate parsing.
