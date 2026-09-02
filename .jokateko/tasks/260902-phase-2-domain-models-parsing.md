+++
title = 'Phase 2: Domain Models & Parsing'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'Domain models, Hugo-standard TOML frontmatter parser, Goldmark AST criteria extraction, and completion formatter.'
dependencies = ['260902-phase-1-configuration-engine']
+++

# Phase 2: Domain Models & Parsing

Domain models and markdown parsing subsystem in `internal/model` and `internal/parser`.

## Acceptance Criteria
- [x] 2.1 Define Domain Models (task, milestone, strategy, glossary, board)
- [x] 2.2 Frontmatter Parser & Delimiter Splitter (+++ TOML)
- [x] 2.3 Markdown Parser & Acceptance Criteria Extractor (Goldmark AST)
- [x] 2.4 Checkbox Item Management & Completion Summary Formatter
- [x] 2.5 Unit Tests for Parsing Engine (parser_test.go)
