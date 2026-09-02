+++
title = 'Phase 5: Validation Engine & DAG Cycle Detection'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'Compiler-style diagnostics, frontmatter invariant rules, 3-color DFS cycle detection (DAG-001), and report formatting.'
dependencies = ['260902-phase-4-atomic-writer-filesystem-watcher']
+++

# Phase 5: Validation Engine & DAG Cycle Detection

Project validation and dependency graph verification in `internal/validator`.

## Acceptance Criteria
- [x] 5.1 Frontmatter Schema Rules Validator (TSK, MLS, STR, GLS)
- [x] 5.2 Dependency Graph DAG & Cycle Detection (DAG-001)
- [x] 5.3 Diagnostic Formatter & Engine Orchestrator
- [x] 5.4 Unit Tests for Validation Engine
