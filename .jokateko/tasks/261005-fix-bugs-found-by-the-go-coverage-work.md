+++
title = 'Fix bugs found by the Go coverage work'
status = 'backlog'
priority = 'medium'
tags = ['bugfix', 'backend']
summary = 'Fix four production bugs that the coverage tasks found and confirmed: an FTS5 query with unbalanced quotes returns a 500, a possible index panic in the validator, wrong `..`-prefix path checks, and `parse -dir` reporting OK for a missing directory.'
created_at = '2026-10-05T06:03:19Z'
changed_at = '2026-10-05T06:03:19Z'
+++

## Context
The 261005 Go coverage tasks (T1–T4) found these bugs. The tests were written without changing production code, so the cases are skipped or written up in notes.

## Acceptance Criteria
- [ ] `internal/store/search.go` `formatFTS5Query`: only pass a quoted query through as-is when the quotes are balanced. Otherwise escape it. A lone `"` or `"a "b"` currently makes `/api/search` return 500 (`unterminated string`). Un-skip `TestSearchUnbalancedQuotePassthrough`.
- [ ] `internal/validator/validator.go:150`: change `len(msg) >= 7 && msg[7] == ':'` to `>= 8`, so a message exactly 7 characters long can't index out of range. Add a test for it.
- [ ] `internal/config/validate.go:208` `validatePathWithinRoot`: use `rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))`, so a folder inside the root such as `..cache` is accepted. Un-skip the `dot-dot_prefixed_directory_name_inside_root` case. Apply the same fix to `isInDir` in `internal/watcher/ingest.go`.
- [ ] `jokateko parse -dir <nonexistent>`: exit non-zero with an error instead of printing "[OK] Validation successful". Decide whether `build -dir <nonexistent>` should also fail rather than creating the directory and exporting an empty board.
- [ ] `go test ./...` passes

## Out of scope / notes
- go-sdk v1.8.0 clients open with `server/discover`, not `initialize`, so the proxy's handshake replay never runs with the current client. Look into whether replay needs to cover the new handshake.
- gentypes turns pointer types into `unknown` and drops embedded struct fields. Neither occurs in internal/model today.
