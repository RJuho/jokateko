# Jokateko File Structure Specification

This document details both the **Source Repository Layout** (where Jokateko itself is developed) and the **Target Project Workspace Layout** (where Jokateko manages tasks inside a user's repository).

---

## 1. Source Repository Layout

```text
/workspaces/jokoteko/
├── cmd/
│   └── jokateko/
│       ├── main.go                     # Root CLI dispatcher & signal handling
│       ├── serve.go                    # 'serve' command implementation
│       ├── parse.go                    # 'parse' / 'lint' command implementation
│       ├── build.go                    # 'build' static export command implementation
│       ├── init.go                     # 'init' project scaffolding implementation
│       ├── mcp.go                      # 'mcp' stdio proxy / standalone command
│       └── version.go                  # 'version' command
├── internal/
│   ├── config/
│   │   ├── config.go                   # Config struct & defaults
│   │   ├── load.go                     # TOML loader (pelletier/go-toml/v2)
│   │   └── config_test.go
│   ├── model/
│   │   ├── task.go                     # Task entity, frontmatter struct, status enum
│   │   ├── milestone.go                # Milestone entity & frontmatter struct
│   │   ├── strategy.go                 # Strategy entity, tier enum
│   │   ├── glossary.go                 # Glossary term entity
│   │   └── board.go                    # Column, Tag, Filter, and Board state models
│   ├── parser/
│   │   ├── frontmatter.go              # YAML frontmatter delimiter splitter & parser
│   │   ├── markdown.go                 # Goldmark AST parser & HTML renderer
│   │   └── parser_test.go
│   ├── store/
│   │   ├── store.go                    # In-memory SQLite connection & mutex wrapper
│   │   ├── schema.sql                  # SQLite DDL tables & indexes
│   │   ├── queries.sql                 # sqlc query definitions
│   │   ├── queries.sql.go              # sqlc generated query code
│   │   ├── models.go                   # sqlc generated models
│   │   └── store_test.go
│   ├── watcher/
│   │   ├── watcher.go                  # fsnotify directory watcher & debounce loop
│   │   └── watcher_test.go
│   ├── writer/
│   │   ├── writer.go                   # Atomic file writer (tmp file + rename)
│   │   └── writer_test.go
│   ├── validator/
│   │   ├── validator.go                # Project validator, DAG dependency checker
│   │   ├── cycle.go                    # Cycle detection algorithm (DFS)
│   │   └── validator_test.go
│   ├── server/
│   │   ├── server.go                   # HTTP server router & listener
│   │   ├── sse.go                      # SSE Hub, client subscriber registry, broadcaster
│   │   ├── handlers_board.go           # Board API handler
│   │   ├── handlers_tasks.go           # Task CRUD handlers
│   │   ├── handlers_milestones.go      # Milestone CRUD handlers
│   │   ├── handlers_strategies.go      # Strategy reading handlers
│   │   ├── handlers_glossary.go        # Glossary reading handlers
│   │   ├── handlers_health.go          # Health check endpoint (/api/health)
│   │   └── server_test.go
│   ├── mcp/
│   │   ├── mcp.go                      # MCP server initialization (Official Go SDK)
│   │   ├── tools_task.go               # list_tasks, get_task, create_task, update_task, complete_task
│   │   ├── tools_milestone.go          # list_milestones, get_milestone, create_milestone
│   │   ├── tools_strategy.go           # list_strategies, get_strategy (Progressive Disclosure)
│   │   ├── tools_glossary.go           # lookup_glossary
│   │   ├── tools_search.go             # search (universal and scoped full-text search)
│   │   ├── tools_tag.go                # list_tags
│   │   ├── tools_board.go              # get_board_state
│   │   ├── resources.go                # MCP resources
│   │   ├── prompts.go                  # MCP prompt templates
│   │   └── mcp_test.go
│   ├── proxy/
│   │   ├── proxy.go                    # Stdio-to-HTTP JSON-RPC proxy
│   │   └── proxy_test.go
│   └── exporter/
│       ├── exporter.go                 # Static HTML snapshot injector & exporter
│       └── exporter_test.go
├── web/
│   ├── src/
│   │   ├── index.tsx                   # Preact SPA entrypoint
│   │   ├── app.tsx                     # Main layout & router
│   │   ├── types/                      # TypeScript definitions (matching Go models)
│   │   │   └── models.ts
│   │   ├── state/                      # Signals / State management (Live vs Static Mode)
│   │   │   ├── store.ts
│   │   │   └── sse.ts
│   │   ├── components/                 # Preact UI components
│   │   │   ├── board/
│   │   │   │   ├── KanbanBoard.tsx
│   │   │   │   ├── Column.tsx
│   │   │   │   └── TaskCard.tsx
│   │   │   ├── modal/
│   │   │   │   ├── TaskDetailModal.tsx
│   │   │   │   └── TaskEditModal.tsx
│   │   │   ├── milestones/
│   │   │   │   └── MilestonesView.tsx
│   │   │   ├── strategies/
│   │   │   │   └── StrategiesView.tsx
│   │   │   ├── glossary/
│   │   │   │   └── GlossaryView.tsx
│   │   │   ├── common/
│   │   │   │   ├── Header.tsx
│   │   │   │   ├── FilterBar.tsx
│   │   │   │   └── Badge.tsx
│   │   ├── styles/
│   │   │   └── input.css               # Tailwind CSS input
│   │   └── index.html                  # HTML template with data injection placeholder
│   ├── dist/                           # Bun build output directory
│   │   ├── index.html
│   │   ├── app.js
│   │   └── app.css
│   ├── embed.go                        # //go:embed dist/* directive
│   ├── package.json                    # Bun configuration
│   ├── bun.lockb                       # Bun lockfile
│   └── tailwind.config.js              # Tailwind CSS configuration
├── tests/
│   └── e2e/                            # Playwright E2E test suite
│       ├── board.spec.ts
│       ├── task-mutation.spec.ts
│       └── static-export.spec.ts
├── docs/                               # Technical specifications
│   ├── README.md
│   ├── module-structure.md
│   ├── file-structure.md
│   ├── config-specification.md
│   ├── mcp-specification.md
│   ├── web-ui-architecture.md
│   └── validation-and-linting.md
├── AGENTS.md                           # AI instructions and rules
├── COMMANDS.md                         # CLI reference documentation
├── Makefile                            # Build automation (bun build, tailwind, go build)
└── go.mod                              # Go module definition
```

---

## 2. Target Project Workspace Layout

When a user or AI agent runs `jokateko init` inside their project, the following `.jokateko/` workspace directory is created:

```text
my-project/
├── .jokateko/
│   ├── config.toml                     # Project configuration (columns, server, paths)
│   ├── tasks/                          # Markdown task tickets
│   │   ├── 260901-setup-database.md
│   │   ├── 260902-auth-endpoints.md
│   │   └── 260903-ui-kanban-board.md
│   ├── milestones/                     # Milestone targets
│   │   ├── 260915-mvp-release.md
│   │   └── 261030-v1-beta.md
│   ├── strategies/                     # Architectural guidelines (Progressive Disclosure)
│   │   ├── architecture.md             # Tier 1 (Core)
│   │   ├── database-rules.md           # Tier 2 (Domain)
│   │   └── frontend-state.md           # Tier 3 (Implementation)
│   └── glossary/                       # Standardized terminology (individual Markdown files)
│       ├── tasks-as-code.md
│       └── progressive-disclosure.md
├── src/                                # User's actual application code
├── AGENTS.md                           # Instructions for AI agents working on this repo
└── README.md
```

### Gitignore Best Practice for Target Repositories

The user's `.gitignore` should include:
```gitignore
# Optional build export directory if generated in project
dist-kanban/
```
All files in `.jokateko/tasks/`, `.jokateko/milestones/`, `.jokateko/strategies/`, and `.jokateko/glossary/` MUST be committed to Git.

> [!NOTE]
> **No Runtime Lockfile Needed:** Jokateko does not create or require a `.jokateko/daemon.json` file. Daemons and CLI proxies detect running instances by probing the configured HTTP health check endpoint (`/api/health`) directly, eliminating stale lockfile issues during crashes.

---

## 3. File Naming Conventions

| Entity | File Path Pattern | Example | Notes |
|---|---|---|---|
| **Task** | `.jokateko/tasks/YYMMDD-short-title.md` | `260901-setup-db.md` | `YYMMDD` prefix ensures chronological sorting; slug is the unique identifier. |
| **Milestone** | `.jokateko/milestones/YYMMDD-short-title.md` | `261015-v1-release.md` | `YYMMDD` indicates target completion date. |
| **Strategy** | `.jokateko/strategies/short-title.md` | `architecture.md` | Tier defined in frontmatter (`tier: 1`, `2`, or `3`). |
| **Glossary** | `.jokateko/glossary/short-title.md` | `tasks-as-code.md` | Individual Markdown files prevent Git merge conflicts; frontmatter holds `tags` and `summary`. |
| **Config** | `.jokateko/config.toml` | `config.toml` | Project configuration. |

---

## 4. Entity File Formats & Examples

### 4.1 Task Markdown File Format (`.jokateko/tasks/YYMMDD-short-title.md`)

```markdown
---
title: "Implement SQLite In-Memory Database Store"
status: "in_progress"
priority: "high"
milestone: "260915-mvp-release"
tags: ["backend", "database", "sqlite"]
summary: "Set up pure Go in-memory SQLite tables using modernc.org/sqlite and sqlc."
dependencies:
  - "260901-project-scaffolding"
---

## Description
We need a pure Go in-memory SQLite store that executes embedded DDL migrations and provides high-performance relational queries without CGO.

## Acceptance Criteria
- [ ] Uses `modernc.org/sqlite` without CGO.
- [ ] Embeds `schema.sql` via `//go:embed`.
- [ ] Safe for concurrent readers and writer via `sync.RWMutex`.
- [ ] Unit tests verify table creation and upsert functionality.
```

### 4.2 Milestone Markdown File Format (`.jokateko/milestones/YYMMDD-short-title.md`)

```markdown
---
title: "MVP Release"
status: "open"
priority: "high"
target_date: "2026-09-15"
tags: ["release", "mvp"]
summary: "Core local daemon with file watcher, in-memory SQLite, and live Preact Kanban board."
---

## Goals
Deliver the first working version of Jokateko capable of running `serve`, watching `.jokateko/tasks/`, and displaying tickets in the browser with real-time SSE updates.
```

### 4.3 Strategy Markdown File Format (`.jokateko/strategies/short-title.md`)

```markdown
---
title: "Zero CGO and Single Executable Architecture"
tier: 1
tags: ["architecture", "go", "dependencies"]
summary: "All Go code must cross-compile cleanly without CGO or external system libraries."
---

# Zero CGO Architecture

## Rule
Under no circumstances may any dependency require CGO or external shared libraries (e.g. `gcc`, `glibc`).

## Rationale
Jokateko distributes as a standalone binary for macOS, Linux, and Windows. Utilizing pure Go dependencies ensures:
1. Instant deterministic cross-compilation (`GOOS=darwin`, `GOOS=windows`, `GOOS=linux`).
2. Frictionless installation for users and CI/CD pipelines.
```

### 4.4 Glossary Markdown File Format (`.jokateko/glossary/short-title.md`)

```markdown
---
title: "Tasks-as-Code"
tags: ["methodology", "core"]
summary: "The methodology where actionable tasks and project specs are stored as version-controlled Markdown files in the repository, making them the absolute source of truth."
---

# Tasks-as-Code

## Concept Overview
The Tasks-as-Code methodology treats tasks, tickets, bugs, and specifications as first-class, version-controlled Markdown documents living inside the Git repository.

## Benefits
1. **Auditable History:** Every ticket change is tied to a Git commit hash.
2. **Offline First:** Developers and AI agents can create, inspect, and update tasks without network access.
3. **No Merge Conflicts:** Distinct task and glossary files ensure parallel work across feature branches merges cleanly.
```
