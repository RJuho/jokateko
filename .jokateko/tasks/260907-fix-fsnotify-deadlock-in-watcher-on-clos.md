+++
title = 'Fix fsnotify deadlock in watcher on close'
status = 'done'
priority = 'high'
tags = []
summary = 'Watcher crashes the process on termination when closing fsnotify.'
created_at = '2026-09-07T08:12:08Z'
changed_at = '2026-09-07T08:22:57Z'
+++

The fsnotify event loop was returning early when closed, but fsWatcher.Close() blocks until the loop is drained. Added event draining for both Events and Errors channels when closed.

## Completion Summary
- **Completed At:** 2026-09-07T08:22:57Z

### What Was Done
Fixed the fsnotify deadlock in watcher.go by draining the Events and Errors channels when the watcher is closed or canceled. Passed all tests, compiled and placed in bin directory.

### Why / Rationale
To prevent the unkillable hang on termination when fsnotify is closed.
