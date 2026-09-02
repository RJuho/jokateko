+++
title = 'Phase 9: Static Build Exporter'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'Snapshot serialization, embedded single-file HTML payload injection, and jokateko build subcommand.'
dependencies = ['260902-phase-8-model-context-protocol-mcp-serve']
+++

# Phase 9: Static Build Exporter

Static snapshot builder and HTML exporter in `internal/exporter`.

## Acceptance Criteria
- [x] 9.1 Snapshot Serializer
- [x] 9.2 HTML Snapshot Injector (jokateko-data script tag replacement)
- [x] 9.3 build Subcommand Implementation
- [x] 9.4 Unit Tests for Exporter
