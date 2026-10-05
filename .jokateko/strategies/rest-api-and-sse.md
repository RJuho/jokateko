+++
title = 'REST API and SSE'
tier = 3
tags = ['api', 'backend', 'security']
summary = 'The HTTP surface of `jokateko serve`: REST endpoints and their payloads, error conventions, the request security pipeline (Host check, Cross-Origin protection, JSON-only bodies, CSP), and the SSE event stream used by the live Web UI.'
+++

# REST API and SSE

Served by `internal/server` on `[server] host:port` (default `127.0.0.1:8080`) with the standard `net/http` mux. The REST API exists for the bundled Web UI. It is **not a stable public API**, and agents use MCP. All bodies are `application/json; charset=utf-8`. Mutations call `internal/service`, the same code as MCP.

## Conventions

- Errors: `{"error": "<message>"}`. `400` invalid input, `404` not found, `409` conflict (duplicate ID, cycle, locked body, delete blocked), `403` rejected Host or cross-origin request, `415` non-JSON body, `500` unexpected.
- Create → `201` with the entity. Update → `200` with the entity. Delete → `200` `{"status":"deleted","id":"…"}`.
- Bodies are capped at 1 MiB.
- `body_html` in every entity is rendered by `internal/parser` in goldmark safe mode: raw HTML becomes escaped text (pure HTML comments are dropped) and dangerous link URLs are emptied, so the UI may insert it with `dangerouslySetInnerHTML`. Never re-enable `html.WithUnsafe()`.

## Endpoints

| Method & path | Notes |
|---|---|
| `GET /api/health` | `{status, project, workspace, version, commit, uptime_seconds}`. `workspace` is the absolute project root, which `jokateko mcp` uses for daemon discovery |
| `GET /api/version` · `/api/about` · `/api/licenses` | Build metadata; About dialog data; bundled license texts |
| `GET /api/board` | `BoardState`: columns with sorted tasks, plus UI config (`editable_states`, `locale`, `translations`, MCP instructions, …). Query: `status`, `milestone`, `tag`, `priority`, `q` |
| `GET /api/tasks` · `GET /api/tasks/{id}` | Same filters as the board |
| `POST /api/tasks` | `{id?, title, summary, status?, priority?, milestone?, tags?, dependencies?, target_at?, body?}` |
| `PUT /api/tasks/{id}` | Partial: any of `title, status, priority, milestone, tags, summary, dependencies, target_at, body`. A `body` change outside `editable_states` → `409` unless only checkboxes changed |
| `PUT /api/tasks/{id}/status` | `{status}`. Any configured column, **including `done`** (humans may skip `complete_task`) |
| `DELETE /api/tasks/{id}[?force=true]` | `409` while other tasks depend on it. `force` leaves dangling dependency IDs |
| `POST /api/tasks/{id}/dependencies` · `DELETE /api/tasks/{id}/dependencies/{depId}` | `{dependency_id}`. Existence and cycle checks |
| `POST /api/tasks/{id}/notes` | `{note}` → appended under `## Notes` |
| `GET/POST /api/milestones`, `GET/DELETE /api/milestones/{id}[?force=true]` | POST `{id?, title, summary, status?, target_date?, tags?, body?}` |
| `GET/POST /api/strategies`, `GET/DELETE /api/strategies/{id}` | POST `{id?, title, tier, summary, tags?, body?}` |
| `GET/POST /api/glossary`, `GET/DELETE /api/glossary/{id}` | POST `{id?, title, summary, tags?, body?}` |
| `GET /api/tags` | `{enforced, tags:[{tag, task_count, milestone_count, strategy_count}]}` |
| `GET /api/search` | `q` (required), `tag`, `type` (`all`/`task`/`milestone`/`strategy`/`glossary`), `limit` (default 20). FTS5 results with snippets |
| `GET/POST/DELETE /api/mcp` | MCP Streamable HTTP (see *MCP server and stdio proxy*) |
| `GET /api/events` | SSE, see below |
| `GET /assets/mermaid-<version>.min.js` | Embedded Mermaid runtime, immutable caching |
| `GET /robots.txt` | `User-agent: *` / `Allow: /` (kept for the Lighthouse SEO audit; the server is loopback-only anyway) |
| `GET /` | The embedded UI (gzip when accepted) |

There is no REST update for milestones, strategies or glossary terms. Those updates exist only as MCP tools. REST does **not** enforce the `[tags]` vocabulary or the archived-milestone guard; MCP does.

## Request security pipeline

Every request passes through, in order:

1. Panic recovery, then the headers `Content-Security-Policy`, `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`. The policy is built by `internal/csp` (`csp.Build`) from `[server.security.csp]` plus the hashes of the inline script and style and the Mermaid SRI hash. `jokateko build` uses the same builder for the `<meta http-equiv>` policy in static exports (see *Web UI architecture*).
2. **Host check (DNS rebinding):** on a loopback connection, `Host` must be `localhost`, `*.localhost`, a loopback IP, or exactly `[server] host`. Otherwise `403`. Connections to a non-loopback address (for example when bound to `0.0.0.0` and reached over the LAN) are **not** restricted by this check.
3. CORS: if `cors_enabled`, matching origins from `cors_allowed_origins` are reflected (an empty list or `"*"` allows any origin). `OPTIONS` → `204`.
4. For `POST`/`PUT` under `/api/` except `/api/mcp`: a declared non-JSON `Content-Type` → `415`, which blocks form and `text/plain` CSRF. A 1 MiB body cap is applied.
5. Go's `http.CrossOriginProtection` rejects cross-site state-changing requests, using `Sec-Fetch-Site`/`Origin`. Configured CORS origins are trusted.

There is **no authentication**. The security model is "loopback only". See SECURITY.md.

## Server-Sent Events: `GET /api/events`

Headers: `text/event-stream`, `no-cache`, `keep-alive`, `X-Accel-Buffering: no`. Frames are `event: <name>\ndata: <json>\n\n`.

| Event | Payload | Sent when |
|---|---|---|
| `connected` | `{}` | On subscribe |
| `ping` | `null` | Every 15 s |
| `task.created` · `task.updated` | Full `Task` | Service mutation, or an external file change (always `.updated`) |
| `milestone.*` · `strategy.*` · `glossary.*` | Full entity | Same as for tasks |
| `<type>.deleted` | `{"id": "…"}` | Deleted by the service or on disk |

Events are **best-effort**. The hub drops an event when its broadcast buffer or a client's 16-event buffer is full, so clients must tolerate gaps. Reloading the board recovers. `board.refreshed` is handled by the client but never emitted.
