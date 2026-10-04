+++
title = 'Go backend review fixes and dependency upgrade'
status = 'done'
priority = 'high'
tags = ['backend', 'dependencies', 'refactor', 'security']
summary = 'Go backend hardened and simplified: REST and MCP share one internal/service mutation layer, path-traversal/CSRF/DNS-rebinding holes are closed, MCP edits reach the Web UI live, MCP runs over stateless Streamable HTTP, and all Go dependencies are on their latest versions.'
created_at = '2026-10-04T19:51:54Z'
changed_at = '2026-10-04T20:22:19Z'
+++

## Context
Review of `internal/` and `cmd/jokateko` found security holes, correctness bugs and heavy duplication between REST handlers and MCP tools. Decision: shared service layer + Streamable HTTP MCP transport.

## Acceptance Criteria
- [x] Go dependencies updated to latest (`go get -u ./... && go mod tidy`), licenses.json regenerated
- [x] Shared helpers: `config.ResolveDirs`, single `Slugify` + `ValidateID`
- [x] `internal/service` layer used by REST and MCP (validation, collision check, write, upsert, SSE notify)
- [x] REST create rejects path-traversal IDs (400); duplicate IDs get suffix / 409
- [x] MCP mutations broadcast SSE events to the Web UI
- [x] HTTP hardening: CrossOriginProtection, Host allow-list, JSON content-type, MaxBytesReader
- [x] MCP daemon transport migrated to Streamable HTTP; proxy verifies workspace via /api/health
- [x] Lifecycle: single signal context, ProcessAll continues past bad files, watcher honours ctx, dead config-reload code removed
- [x] Cleanups: gofmt (lint enforces), small modernisations
- [x] `go vet`, staticcheck, `make test`, `make e2e-test` pass; `make install` done

## Notes

### [2026-10-04 20:09 UTC]

Implementation summary (awaiting human review, not committed):

- **Deps:** go-sdk 1.7.0→1.8.0, goldmark 1.8.6, sqlite 1.57.0→1.60.1 (libc 1.77.1), x/* bumps; licenses.json regenerated.
- **New `internal/service`:** single mutation path for REST + MCP (ID validation `^[a-z0-9][a-z0-9-]{0,99}$`, collision handling: explicit → 409, generated → `-2` suffix; serialized mutations; SSE notify). Net diff: −1.7k lines.
- **Security:** `http.CrossOriginProtection` (CORS origins trusted), loopback Host check (DNS rebinding), JSON-only POST/PUT, 1 MiB body cap.
- **MCP transport:** Streamable HTTP in **stateless** mode — required because go-sdk 1.8 clients default to protocol 2026-07-28 (sessionless), which stateful servers reject. Proxy uses `DisableStandaloneSSE`, verifies `/api/health.workspace`.
- **Proxy bug found by tests:** backend→client pump treated read errors from the just-closed previous backend as a daemon failure and flipped back to standalone; now ignored when the backend changed.
- **Behaviour changes to review:** REST create/update now reject statuses that are not board columns (fixtures used `todo`); MCP `allow_mutations=false` now blocks all write tools (previously only some); config hot-reload stub removed (was never wired).
- **After install:** restart any running `jokateko serve` daemon — the new `jokateko mcp` proxy speaks Streamable HTTP and won't attach to an old SSE daemon (it falls back to standalone mode).
- Not done (out of scope): staticcheck findings in cmd/genlicenses and cmd/gentypes; `go test -race` unavailable (no C toolchain in container).

## Completion Summary
- **Completed At:** 2026-10-04T20:22:19Z

### What Was Done
- Dependencies: go-sdk 1.7.0→1.8.0, goldmark 1.8.6, modernc sqlite 1.57.0→1.60.1 (libc 1.77.1), golang.org/x/* bumps; licenses.json regenerated.
- New internal/service package: create/update/delete for tasks, milestones, strategies and glossary terms; ID validation (lowercase slug regex), ID allocation (explicit duplicate → ErrConflict/409, generated → -2/-3 suffix), serialized mutations, SSE notification via a Notifier interface, typed error kinds (ErrInvalid/ErrNotFound/ErrConflict) mapped to HTTP status codes. REST handlers and MCP tools are thin adapters (net −1.7k lines).
- HTTP hardening: http.CrossOriginProtection (configured CORS origins trusted), loopback Host-header check, JSON-only POST/PUT, 1 MiB body limit, Vary headers.
- MCP: Streamable HTTP handler in stateless mode (mcp.Server.HTTPHandler); proxy uses StreamableClientTransport, verifies /api/health workspace, probes loopback cfg.Server.Host; fixed pump race that flipped back to standalone after a backend switch.
- Lifecycle: signal.NotifyContext in main threaded to serve/mcp; watcher.StartPipeline shared by serve and proxy; ProcessAll continues past malformed files (errors.Join); watcher loop honours ctx and reuses its timer; removed unwired config-reload code.
- Helpers/cleanups: config.ResolveDirs/Dirs, config.Columns, single Slugify, writer fsyncs parent dir, SuppressionCache plain Mutex, errors.Is(fs.ErrNotExist), sync.OnceValues, staticcheck/modernize fixes, repo-wide gofmt, gofmt check in make lint.
- Tests: service unit tests; server security tests (traversal, duplicate ID, cross-site, Host, content type, body size); end-to-end MCP→SSE test; ProcessAll resilience; proxy workspace verification. Docs updated (API, MCP, module and file structure).

### Why / Rationale
Most bugs (missing SSE broadcasts, ignored errors, unvalidated IDs, silent overwrites) came from REST and MCP each re-implementing the same persistence workflow, so a single service layer fixes them at the root and keeps both interfaces consistent. The daemon is a local tool with write access to the repository, so cross-origin and DNS-rebinding protection are needed even on 127.0.0.1; standard-library CrossOriginProtection and the SDK's localhost protection avoid new dependencies. Stateless Streamable HTTP is required because go-sdk 1.8 clients default to the sessionless 2026-07-28 protocol, and the deprecated SSE transport does not support it. Workspace verification prevents an agent from silently editing another project's daemon on the same port.
