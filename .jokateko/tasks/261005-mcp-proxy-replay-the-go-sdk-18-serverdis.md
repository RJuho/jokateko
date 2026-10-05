+++
title = 'MCP proxy: replay the go-sdk 1.8 server/discover handshake on failover'
status = 'done'
priority = 'medium'
tags = ['api', 'backend', 'bugfix']
summary = "The stdio proxy now replays a handshake only where it is needed: the legacy `initialize` handshake, when switching to the stateful standalone engine. It uses a private request ID and never forwards the replay's output to the client. SEP-2575 clients (Claude Code 2.1.289, go-sdk 1.8), which open with `server/discover`, and every switch to the stateless daemon need no replay."
created_at = '2026-10-05T06:18:21Z'
changed_at = '2026-10-05T06:29:53Z'
+++

## Context
Found during the Go coverage work (task 261005-go-test-coverage-t2-store-server-validat).

- `internal/proxy/proxy.go` `inspectClientMessage` caches only `initialize` and `notifications/initialized`. `replayHandshake` writes those to the new backend in `switchToProxy` and `switchToStandalone`, and reads and discards one response.
- go-sdk v1.8.0 adds `methodDiscover = "server/discover"` (`mcp/protocol.go:2325`, client `mcp/client.go:429`, server handler `mcp/server.go:920`). The server fills in the initialize state from a discover call (`server.go:1978`). A go-sdk 1.8 client in the coverage tests opened with `server/discover`, so `cachedInitMsg` stayed nil and the replay was a no-op.
- The daemon side uses stateless Streamable HTTP, which may not need a replay at all. The local standalone engine is a stateful in-process MCP server over pipes, and it probably does need one.

## Acceptance Criteria
- [x] Investigate and document which handshake real clients send (Claude Code, go-sdk 1.8 client, an older `initialize`-only client), and whether each backend (stateless daemon over HTTP, stateful standalone engine) needs a replay after a switch
- [x] `inspectClientMessage` also caches the `server/discover` request when the investigation shows it is needed. Keep `initialize` support for older clients.
- [x] `replayHandshake` replays whichever handshake was cached, in the original order. Replay responses are consumed and never forwarded to the client. Request IDs must not clash with in-flight client requests.
- [x] Test: a go-sdk 1.8 client connected through the proxy keeps working (`tools/list`, `tools/call`) after daemon → standalone and standalone → daemon switches
- [x] Test: the existing `initialize`-based replay keeps working
- [x] Proxy behaviour is documented in docs/ (handshake replay section)
- [x] `go test ./...` passes

## Notes

### [2026-10-05 06:26 UTC]

**Investigation (go-sdk v1.8.0, Claude Code 2.1.289)**

- **Claude Code 2.1.289**, captured with a `tee` shim in front of `jokateko mcp`:
  - Opens with `server/discover` (protocol `2026-07-28`, id `"server-discover-probe-1"`), then sends `subscriptions/listen` (id `"listen:0"`).
  - Every later request carries `_meta` with the protocol version, client info and client capabilities.
  - It sends no `initialize`.
- **go-sdk 1.8 `Client`**: does the same (discover first, `_meta` on every request). It falls back to `initialize` only if the server rejects discover.
- **Server side** (`server.go` `ServerSession.handle`): a request that carries the SEP-2575 `_meta` skips the "initialized" check, and the server fills in session state from that `_meta`. So a fresh backend serves SEP-2575 clients without any replay. Replaying `server/discover` is not needed, so it is not cached.
- **Daemon**: stateless Streamable HTTP. It makes up initialize state for old-protocol requests (`streamable.go` around line 503), so it never needs a replay. `switchToProxy` no longer replays.
- **Standalone engine**: stateful. It needs a replay only for legacy `initialize` clients; without one, calls fail with `method "tools/call" is invalid during session initialization`. Confirmed by temporarily disabling the replay: `TestProxy_LegacyInitializeSurvivesSwitches` then fails.

**Changes made**
- The replay now sends `initialize` under a private ID (`jokateko-proxy-replay-<n>`) and reads until the response with that ID arrives, waiting at most 5 s. Other messages are skipped and nothing is forwarded to the client. An error response fails the switch.
- `inspectClientMessage` now uses a type switch on `*jsonrpc.Request`.
- Docs: `docs/mcp-specification.md` §2.3.

**Known limitation (documented)**: a `subscriptions/listen` stream is not carried over to the new backend after a switch. Harmless, because all tools, prompts and resources are registered once in `mcp.New`, so no list-changed notifications are ever sent.

**Not verified**: the race detector could not run, because there is no gcc in the container. `go test ./internal/proxy -count=10` passes instead.

## Completion Summary
- **Completed At:** 2026-10-05T06:29:53Z

### What Was Done
- `internal/proxy/proxy.go`:
  - `replayHandshake` sends the cached `initialize` with a proxy-private ID (`jokateko-proxy-replay-<n>`, from an atomic counter). It reads until the response with that ID arrives, waiting at most 5 s, and skips anything else read before it. An error response fails the switch. It then sends the cached `notifications/initialized`.
  - `inspectClientMessage` uses a type switch on `*jsonrpc.Request` and caches only `initialize` calls and `notifications/initialized`.
  - `switchToProxy` no longer replays.
- Tests:
  - New `internal/proxy/handshake_test.go`. A go-sdk 1.8 client, checked to open with `server/discover`, and a raw `initialize`-only client both switch standalone → daemon → standalone, with tool calls checked after each switch. The raw-client test also checks that no unexpected response reaches the client.
  - `internal_test.go`: scripted `fakeConn` replies, with cases for the private ID, skipping messages, rejection, timeout, unique IDs, and that `server/discover` is not cached.
- Docs: `docs/mcp-specification.md` §2.3, "Handshake Replay on Failover".

### Why / Rationale
- A request that carries the SEP-2575 `_meta` sets up session state on its own (go-sdk `ServerSession.handle`), so replaying `server/discover` would add nothing.
- The daemon is stateless Streamable HTTP and makes up initialize state for every request, so a replay there only adds a round trip and a way to fail.
- The standalone engine is a stateful server, so older `initialize`-only clients still need the replay.
- The private ID avoids clashing with client IDs (Claude Code uses string IDs such as `"listen:0"`).
- Matching the response by ID fixes the old read-one-message assumption, which could have leaked the real response to the client.
- `subscriptions/listen` is not carried over to the new backend. This is documented as harmless, because tools, prompts and resources are fixed once the server starts.
