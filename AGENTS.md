# Project name: jokateko

Other documentation:
- README.md (CLI reference)
- Architecture and specifications are Jokateko strategies and glossary terms (`list_strategies`, `lookup_glossary`); start with the tier-1 `architecture` strategy

More resources:
- [Bun documentation](https://bun.com/llms.txt)
- [Valibot documentation](https://valibot.dev/llms.txt)
- [daisyUI documentation](https://daisyui.com/llms.txt)

Remember to check SKILLS (restore them with `make skills` after cloning) and inspect the active MCP server to see what tools are available and review their schemas.

## Workflow
*   Follow given task, document key details to Jokateko Tasks, or if suitable Strategies or Glossary. Check Jokateko MCP tools for Tasks, Strategies and Glossary.
*   Do not make a git commit by yourself, everything must be validated by human.
*   When you are done, leave the task in the 'In Review' (`in_review`) column to wait for human check. Also build and install Jokateko with `make install` (installs to `INSTALL_DIR`, default `~/.local/bin`), so the MCP server picks up your changes.

## 1. Project Overview
This project is a local, Markdown-driven Kanban and task management tool designed for both human developers and AI agents. It operates on a "Tasks-as-Code" and "Spec-First" methodology, where the repository files are the absolute source of truth. The architecture relies on a local backend that handles safe file writing and provides structured Model Context Protocol (MCP) interfaces to AI agents.

## 2. Strict Dependencies (No Exceptions)
Agents MUST NOT introduce any new libraries, frameworks, or dependencies without explicit human approval. The technology stack is locked to the following:
*   **Backend (Go):** Pure Go implementation. C-language dependencies (CGO) are strictly forbidden to ensure cross-platform compilation.
*   **Configuration:** TOML (`github.com/pelletier/go-toml/v2`).
*   **Database & File Watching:** `modernc.org/sqlite` (no CGO) + `sqlc`, and `fsnotify` for unidirectional file tracking.
*   **Markdown Parsing:** `yuin/goldmark`.
*   **AI / MCP Integration:** Official Model Context Protocol Go SDK (`modelcontextprotocol/go-sdk`).
*   **Frontend (JS):** Preact components and Valibot schema validation (`valibot`), bundled into a single self-contained HTML file using the Bun 1.4 native bundler. Styling uses Tailwind CSS.
*   **Communication:** Server-Sent Events (SSE) for Go-to-browser updates, and native `fetch()` in the browser.

## 3. Build Process and Platform Support
The software is distributed as an auditable, zero-dependency single executable across macOS, Linux, and Windows. 
1.  **Web UI (Bun):** The frontend (HTML + inlined CSS + inlined JS) is compiled into a single self-contained `web/dist/index.html` file using Bun.
2.  **Go Binary:** The Go compiler utilizes the `//go:embed` feature to bake the compiled web UI directly into the final backend executable.

## 4. Testing and Deterministic Validation
AI-generated code is subjected to deterministic automated feedback loops.
*   **Go Backend:** Write tests using the standard `testing` library. Run validations via `go test ./...`.
*   **Web UI (E2E):** The UI is validated using Playwright. Agents must utilize `data-testid` attributes in the DOM to ensure reliable targeting.
*   **Configuration:** TOML configuration files must be validated using the `parse` / `lint` command to ensure schema compliance.

## 5. Jokateko MCP & Mutation Guidelines (Crucial)
AI agents MUST NOT perform raw file edits, creations, or deletions directly inside the `.jokateko/` repository directory (such as `.jokateko/tasks/`, `.jokateko/milestones/`, `.jokateko/strategies/`, or `.jokateko/glossary/`).

**Tool Discovery:** Always check with the active MCP server (or MCP environment tools list) to discover what tools are available and inspect their latest parameter schemas.

All operations on tasks, milestones, strategies, and glossary terms MUST be executed exclusively through the provided Jokateko Model Context Protocol (MCP) tools:
*   **Tasks:** `create_task`, `get_task`, `list_tasks`, `update_task_content`, `update_task_status`, `list_task_items`, `update_task_item`, `add_task_note`, `set_task_target`, `complete_task`, `delete_task`
*   **Dependencies:** `add_task_dependency`, `remove_task_dependency`
*   **Milestones:** `create_milestone`, `get_milestone`, `list_milestones`, `update_milestone`, `delete_milestone`
*   **Strategies:** `create_strategy`, `get_strategy`, `list_strategies`, `update_strategy`, `delete_strategy`
*   **Glossary:** `create_glossary_term`, `lookup_glossary`, `update_glossary_term`, `delete_glossary_term`
*   **Discovery & Search:** `get_board_state`, `list_tags`, `search_tasks`, `search_milestones`, `search_strategies`, `search_glossary`, `search_all`

**Why:** The Jokateko backend guarantees atomic file persistence, cyclic dependency checks, automatic slugification, frontmatter validation, cache suppression, and Server-Sent Events (SSE) broadcasting to the live Web UI. Direct filesystem mutations bypass these safety mechanisms and will cause desynchronization or corrupt repository state.