+++
title = 'Phase 0: Project Initialization & Tooling Setup'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'docs']
summary = 'Initial Go module scaffolding, locked zero-CGO backend dependencies, version metadata, and root build pipeline.'
+++

# Phase 0: Project Initialization & Tooling Setup

Foundational scaffolding and tooling for the Jokateko project.

## Acceptance Criteria
- [x] 0.1 Initialize Go Module (github.com/RJuho/jokateko, Go 1.27.0)
- [x] 0.2 Declare Locked Backend Dependencies (go-toml, modernc.org/sqlite, fsnotify, goldmark, mcp-go-sdk)
- [x] 0.3 Scaffold Base Directory Structure and web/embed.go
- [x] 0.4 Create Root Makefile & Code Generation Setup (sqlc, test, build, ui-build)
