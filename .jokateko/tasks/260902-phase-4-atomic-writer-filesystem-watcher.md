+++
title = 'Phase 4: Atomic Writer & Filesystem Watcher'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'Crash-safe atomic file writing with fsync, echo suppression caching, fsnotify debounce loop, and store ingestion.'
dependencies = ['260902-phase-3-storage-in-memory-index']
+++

# Phase 4: Atomic Writer & Filesystem Watcher

Filesystem persistence and change detection in `internal/writer` and `internal/watcher`.

## Acceptance Criteria
- [x] 4.1 Safe Atomic File Persistence with tempfile + rename
- [x] 4.2 Watcher Suppression Cache (Echo Prevention)
- [x] 4.3 Filesystem Watcher & Debounce Loop (fsnotify)
- [x] 4.4 Watcher Ingestion Pipeline
- [x] 4.5 Unit Tests for Writer & Watcher
