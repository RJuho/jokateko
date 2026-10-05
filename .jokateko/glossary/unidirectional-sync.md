+++
title = 'Unidirectional sync'
tags = ['backend']
summary = 'Data flows one way, from the Markdown files to the in-memory SQLite index to the clients, so the files are always the truth and the index can be thrown away and rebuilt at any time.'
+++

```text
.jokateko/*.md ──fsnotify──▶ parser ──▶ in-memory SQLite ──SSE──▶ browsers
      ▲                                                    │
      └──────── service: atomic write ◀── REST / MCP ◀─────┘
```

Writes never update the index first. They write the file atomically, then re-index from the written entity. External edits (an editor, `git pull`, a branch switch) take the same ingest path. The index is never persisted, and it is rebuilt by scanning all files on every start.

Note: `config.toml` is not watched. It is read once at startup.
