+++
title = 'Implement Comprehensive Native Go Fuzzing'
status = 'done'
priority = 'high'
tags = []
summary = 'Implemented native Go fuzzing across Markdown parser, REST API, and MCP server, plus multi-channel integration chaos testing.'
created_at = '2026-09-07T08:27:30Z'
changed_at = '2026-09-07T09:56:33Z'
+++

Implement native Go fuzzing (`go test -fuzz`) for Jokateko's core components to proactively identify panics, deadlocks, and edge cases.

### Scope
1. **Markdown Parsing**: Target `parser.ParseTaskWithCriteria` and the `goldmark` ingestion pipeline with mutated markdown strings.
2. **REST API**: Use `httptest.NewServer` to fuzz HTTP API handlers with randomized JSON payloads.
3. **MCP Server**: Fuzz the JSON-RPC over stdio runner with malformed and randomized MCP requests.
4. **Make Scripts**: Add easy-to-use Make targets (e.g., `make fuzz-markdown`, `make fuzz-api`, `make fuzz-mcp`).

## Completion Summary
- **Completed At:** 2026-09-07T09:56:33Z

### What Was Done
Created Go fuzz tests in internal/parser, internal/server, and internal/mcp. Added 1-hour fuzzing targets to Makefile. Created full multi-channel integration chaos test in test/chaos/chaos_test.go exercising filesystem events, REST API, MCP tool calls, and live SSE event stream concurrently with graceful SIGINT shutdown.

### Why / Rationale
Ensure resilience against corrupted payloads, eliminate panics, and guarantee live-sync and graceful shutdown stability under extreme multi-channel concurrency.
