+++
title = 'Phase 6: CLI Implementation'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'CLI commands dispatcher, init scaffolding, parse/lint validator, version diagnostics, and signal handling.'
dependencies = ['260902-phase-5-validation-engine-dag-cycle-dete']
+++

# Phase 6: CLI Implementation

Command line interface in `cmd/jokateko/`.

## Acceptance Criteria
- [x] 6.1 Root CLI Dispatcher & Flag Handling
- [x] 6.2 version Subcommand with link-time flags
- [x] 6.3 init Scaffolding Subcommand with starter assets
- [x] 6.4 parse / lint Subcommand
- [x] 6.5 Integration Tests for CLI Commands
