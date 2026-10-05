+++
title = 'Move Go chaos test from test/chaos to tests/chaos'
status = 'done'
priority = 'low'
tags = ['housekeeping', 'tests']
summary = 'The Go chaos test now lives in tests/chaos/ with the e2e and lighthouse suites, and the stray top-level test/ folder is gone.'
created_at = '2026-10-05T10:35:26Z'
changed_at = '2026-10-05T10:37:55Z'
+++

## Acceptance Criteria
- [x] `test/chaos/chaos_test.go` moved to `tests/chaos/`, empty `test/` removed
- [x] Makefile `chaos-test` target points at `./tests/chaos`
- [x] CI E2E change filter narrowed to `tests/(e2e|lighthouse)/` so Go-only chaos edits don't trigger Playwright
- [x] docs/file-structure.md and project-structure strategy updated
- [x] `go test ./...` and `make chaos-test` pass

## Notes

### [2026-10-05 10:36 UTC]

Moved with `git mv`. The folder depth is the same, so the test's `../../cmd/jokateko` build path still works. `go vet ./...` and `go test ./...` are clean, and `make chaos-test` passes in 11.7s. I left the old `test/chaos` mentions in the historical task notes as they are. Installed with `make install`.

## Completion Summary
- **Completed At:** 2026-10-05T10:37:55Z

### What Was Done
Moved test/chaos/chaos_test.go to tests/chaos/ with git mv and removed the empty test/ folder. Changed the Makefile chaos-test target to ./tests/chaos. Narrowed the CI E2E change filter in ci.yml from `tests/` to `tests/(e2e|lighthouse)/`. Added chaos/ to docs/file-structure.md and updated the project-structure strategy diagram.

### Why / Rationale
Putting all test suites under one tests/ folder keeps the repo root clean. The folder depth didn't change, so the test's relative build path still works. Without the narrower CI filter, changes to the Go-only chaos test would start Playwright runs they don't need.
