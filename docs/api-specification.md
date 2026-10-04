# Jokateko REST API & Server-Sent Events Specification

This document specifies the HTTP endpoints, data contracts, and Server-Sent Events (SSE) protocol provided by the Jokateko HTTP server daemon (`jokateko serve`).

---

## 1. Design Principles

- **Zero-CGO & Pure Go:** Handled entirely by Go's standard library `net/http` router (`ServeMux`) with path-value extraction (`GET /api/tasks/{id}`).
- **Tasks-as-Code Mutability:** All mutating requests (`POST`, `PUT`, `DELETE`) write markdown files atomically to the filesystem using `internal/writer`, update the in-memory SQLite store, and broadcast live SSE events.
- **Strict Content Negotiation:** API endpoints produce and consume `application/json; charset=utf-8`.
- **Generated Type-Safety:** TypeScript models are generated directly from Go models via `cmd/gentypes` into `web/src/types/generated.ts` and validated at runtime in the UI with Valibot.

---

## 2. Health & System Endpoints

### `GET /api/health`
Returns runtime daemon status, uptime, project name, and version info.
- **Response (200 OK):**
  ```json
  {
    "status": "ok",
    "project": "My Project",
    "workspace": "/home/dev/my-project",
    "version": "v1.1.0",
    "commit": "abc1234",
    "uptime_seconds": 3600
  }
  ```
- `workspace` is the absolute project root. `jokateko mcp` only proxies to a daemon whose `workspace` matches its own directory, so a daemon for another project on the same port is ignored.

### `GET /api/version`
Returns detailed build and platform metadata.
- **Response (200 OK):**
  ```json
  {
    "version": "v1.1.0",
    "commit": "abc1234",
    "date": "2026-09-01T12:00:00Z",
    "go_version": "go1.24.0",
    "platform": "linux/amd64"
  }
  ```

---

## 3. Board & Task Endpoints

### `GET /api/board`
Returns the aggregated board state organized into columns with tasks and item counts.
- **Query Parameters:**
  - `status` (string, optional): Filter by column ID.
  - `milestone` (string, optional): Filter by milestone slug.
  - `tag` (string, optional): Filter by tag.
  - `priority` (string, optional): Filter by priority (`low`, `medium`, `high`, `critical`).
  - `q` (string, optional): Full-text search query.
- **Response (200 OK):** `BoardState` JSON object.

### `GET /api/tasks`
Lists tasks matching optional filter parameters.
- **Query Parameters:** Same as `/api/board`.
- **Response (200 OK):** Array of `Task` objects.

### `GET /api/tasks/{id}`
Returns a single task by ID.
- **Response:**
  - `200 OK`: `Task` object.
  - `404 Not Found`: `{"error": "task not found"}`.

### `POST /api/tasks`
Creates a new task. Writes `.jokateko/tasks/YYMMDD-slug.md`, indexes it in SQLite, and emits `task.created` SSE event.
- **Request Body:**
  ```json
  {
    "id": "260901-my-task", // optional; auto-generated if omitted
    "title": "Implement Login",
    "status": "ready",
    "priority": "high",
    "milestone": "260915-mvp",
    "tags": ["backend", "auth"],
    "dependencies": [],
    "summary": "Short summary",
    "body": "## Acceptance Criteria\n- [ ] ..."
  }
  ```
- **Response (201 Created):** Created `Task` object.

### `PUT /api/tasks/{id}`
Updates an existing task frontmatter and body. Emits `task.updated` SSE event.
- **Request Body:** Partial or complete task update fields.
- **Response (200 OK):** Updated `Task` object.

### `DELETE /api/tasks/{id}`
Removes the task markdown file, deletes it from SQLite, and emits `task.deleted` SSE event.
- **Response (200 OK):** `{"status": "deleted", "id": "..."}`.

---

## 4. Entity CRUD Endpoints

### Milestones
- `GET /api/milestones`: List all milestones with progress statistics.
- `GET /api/milestones/{id}`: Get milestone details.
- `POST /api/milestones`: Create milestone markdown file. Emits `milestone.created`.
- `DELETE /api/milestones/{id}`: Delete milestone. Emits `milestone.deleted`.

### Strategies
- `GET /api/strategies`: List architectural strategies.
- `GET /api/strategies/{id}`: Get strategy details.
- `POST /api/strategies`: Create strategy markdown file.
- `DELETE /api/strategies/{id}`: Delete strategy.

### Glossary
- `GET /api/glossary`: List glossary terms.
- `GET /api/glossary/{id}`: Get glossary term details.
- `POST /api/glossary`: Create glossary markdown file.
- `DELETE /api/glossary/{id}`: Delete glossary term.

---

## 5. Analytics & Search Endpoints

### `GET /api/tags`
Returns tag frequency distribution across all entities.
- **Response (200 OK):**
  ```json
  {
    "enforced": false,
    "tags": [
      {
        "tag": "backend",
        "task_count": 5,
        "milestone_count": 1,
        "strategy_count": 2
      }
    ]
  }
  ```

### `GET /api/search`
Executes SQLite FTS5 full-text search across all entities.
- **Query Parameters:**
  - `q` (string, required): Search query keywords.
  - `tag` (string, optional): Filter by tag.
  - `type` (string, optional): Filter entity type (`all`, `task`, `milestone`, `strategy`, `glossary`).
  - `limit` (int, optional): Max results (default 20).
- **Response (200 OK):** Array of `SearchResult` objects with highlighted snippets.

---

## 6. Server-Sent Events (SSE) Protocol

### `GET /api/events`
Persistent HTTP connection streaming live JSON events to connected Web UI clients.

#### Headers
```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

#### Event Format
Each message conforms to standard SSE protocol:
```http
event: task.updated
data: {"id":"260901-sample","title":"...","status":"done",...}

```

#### Event Catalog
| Event Name | Payload | Trigger |
|---|---|---|
| `connected` | `{}` | Emitted immediately upon client handshake |
| `ping` | `{}` | Periodic heartbeat ping emitted every 15s |
| `task.created` | `Task` | New task created via API or filesystem |
| `task.updated` | `Task` | Task frontmatter, status, or body changed |
| `task.deleted` | `{"id": "..."}` | Task removed from workspace |
| `milestone.created` | `Milestone` | Milestone created |
| `milestone.updated` | `Milestone` | Milestone updated |
| `milestone.deleted` | `{"id": "..."}` | Milestone removed |
| `board.refreshed` | `{}` | Global invalidate requesting full client re-sync |
