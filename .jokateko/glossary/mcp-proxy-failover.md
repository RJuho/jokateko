+++
title = 'MCP proxy failover'
tags = ['backend']
summary = '`jokateko mcp` keeps one stdio session for the agent but switches its backend live: it proxies to a running same-workspace `serve` daemon, or runs an in-process standalone server, checking every 5 s.'
+++

This means you can start, stop or upgrade `jokateko serve` while an agent is connected, and the agent never notices:

- **Proxy mode:** a daemon answers `GET /api/health` with the same `workspace`. JSON-RPC is forwarded to its `/api/mcp`, so there is one store and one watcher, and browsers see agent changes live.
- **Standalone mode:** no daemon. The proxy runs its own store, watcher and MCP server in-process.
- When it switches to standalone, legacy clients' `initialize` handshake is replayed under a private ID. Newer `server/discover` clients need no replay.

Details: strategy *MCP server and stdio proxy*.
