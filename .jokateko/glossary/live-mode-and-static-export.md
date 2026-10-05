+++
title = 'Live mode and static export'
tags = ['frontend']
summary = "The Web UI's operating modes: live (served by `jokateko serve`, editable, updated over SSE), static (a read-only single-file snapshot from `jokateko build`), and client (browser-only, saved in localStorage)."
+++

All three modes use the same `index.html` bundle. The mode is chosen at startup:

| Mode | When | Data | Editing |
|---|---|---|---|
| **Static** | The page contains an embedded snapshot | Snapshot in `<script id="jokateko-data">` | Read-only |
| **Live** | Served over http(s) without a snapshot | REST + SSE from the daemon | Yes, written to disk |
| **Client** | Opened without a server or snapshot | localStorage | Yes, browser only |

A static export (`jokateko build -out board.html`) contains the **full project data**. Share it like you would share the repository. Mermaid can load from a pinned CDN, be bundled for offline use, or be left out (`--mermaidjs=cdn|bundled|none`).

See *Snapshot injection* and the strategy *Web UI architecture*.
