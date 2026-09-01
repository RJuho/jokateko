# Project name: jokateko

Other documentation:
- COMMANDS.md
- /docs/README.md

## 1. Project Overview
This project is a local, Markdown-driven Kanban and task management tool designed for both human developers and AI agents. It operates on a "Tasks-as-Code" and "Spec-First" methodology, where the repository files are the absolute source of truth. The architecture relies on a local backend that handles safe file writing and provides structured Model Context Protocol (MCP) interfaces to AI agents.

### Current and Upcoming Features
*   **Tasks/Tickets:** Core markdown files representing actionable work.
*   **Milestones:** Grouping tasks into larger deliverable targets.
*   **Tags:** Categorization and filtering for tasks and milestones.
*   **Strategies:** Tiered architectural guidelines and rules using a "Progressive Disclosure" pattern.
*   **Glossary:** Standardized project terminology definitions.

## 2. Strict Dependencies (No Exceptions)
Agents MUST NOT introduce any new libraries, frameworks, or dependencies without explicit human approval. The technology stack is locked to the following:
*   **Backend (Go):** Pure Go implementation. C-language dependencies (CGO) are strictly forbidden to ensure cross-platform compilation.
*   **Configuration:** TOML (`github.com/pelletier/go-toml/v2`).
*   **Database & File Watching:** `modernc.org/sqlite` (no CGO) + `sqlc`, and `fsnotify` for unidirectional file tracking.
*   **Markdown Parsing:** `yuin/goldmark`.
*   **AI / MCP Integration:** Official Model Context Protocol Go SDK (`modelcontextprotocol/go-sdk`).
*   **Frontend (JS):** Preact components and Valibot schema validation (`valibot`), bundled into a single self-contained HTML file using the Bun 1.4 native bundler. Styling uses Tailwind CSS Standalone CLI (no Node.js/NPM dependencies).
*   **Communication:** Server-Sent Events (SSE) for Go-to-browser updates, and native `fetch()` in the browser.

## 3. Build Process and Platform Support
The software is distributed as an auditable, zero-dependency single executable across macOS, Linux, and Windows. 
1.  **Web UI (Bun):** The frontend (HTML + inlined CSS + inlined JS) is compiled into a single self-contained `dist/index.html` file using Bun.
2.  **Go Binary:** The Go compiler utilizes the `//go:embed` feature to bake the compiled web UI directly into the final backend executable.

## 4. Testing and Deterministic Validation
AI-generated code is subjected to deterministic automated feedback loops.
*   **Go Backend:** Write tests using the standard `testing` library. Run validations via `go test ./...`.
*   **Web UI (E2E):** The UI is validated using Playwright. Agents must utilize `data-testid` attributes in the DOM to ensure reliable targeting.
*   **Configuration:** TOML configuration files must be validated using the `parse` / `lint` command to ensure schema compliance.