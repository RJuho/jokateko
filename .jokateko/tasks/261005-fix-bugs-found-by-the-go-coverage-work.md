+++
title = 'Fix bugs found by the Go coverage work'
status = 'done'
priority = 'medium'
tags = ['backend', 'bugfix']
summary = 'Fixed the four bugs found by the coverage work: FTS5 malformed-quote 500, validator rule-ID out-of-range, `..`-prefix path checks, and parse/build accepting a missing -dir.'
created_at = '2026-10-05T06:03:19Z'
changed_at = '2026-10-05T06:17:58Z'
+++

## Context
The 261005 Go coverage tasks (T1–T4) found these bugs. The tests were written without changing production code, so the cases are skipped or written up in notes.

## Acceptance Criteria
- [x] `internal/store/search.go` `formatFTS5Query`: only pass a quoted query through as-is when the quotes are balanced. Otherwise escape it. A lone `"` or `"a "b"` currently makes `/api/search` return 500 (`unterminated string`). Un-skip `TestSearchUnbalancedQuotePassthrough`.
- [x] `internal/validator/validator.go:150`: change `len(msg) >= 7 && msg[7] == ':'` to `>= 8`, so a message exactly 7 characters long can't index out of range. Add a test for it.
- [x] `internal/config/validate.go:208` `validatePathWithinRoot`: use `rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))`, so a folder inside the root such as `..cache` is accepted. Un-skip the `dot-dot_prefixed_directory_name_inside_root` case. Apply the same fix to `isInDir` in `internal/watcher/ingest.go`.
- [x] `jokateko parse -dir <nonexistent>`: exit non-zero with an error instead of printing "[OK] Validation successful". Decide whether `build -dir <nonexistent>` should also fail rather than creating the directory and exporting an empty board.
- [x] `go test ./...` passes

## Out of scope / notes
- go-sdk v1.8.0 clients open with `server/discover`, not `initialize`, so the proxy's handshake replay never runs with the current client. Look into whether replay needs to cover the new handshake.
- gentypes turns pointer types into `unknown` and drops embedded struct fields. Neither occurs in internal/model today.

## Notes

### [2026-10-05 06:14 UTC]

Implementation (2026-10-05):
- **FTS5** (`internal/store/search.go`): a quoted query is passed through only when it is a single well-formed phrase: length ≥ 2, inner text not blank, no embedded `"`. Anything else falls through to the existing sanitizer, which strips quotes and ORs prefix terms. The test is renamed to `TestSearchMalformedQuotes` (un-skipped; covers `"`, `""`, `" "`, `"a "b"`, `"Task"x"`). New table cases were added to `TestFormatFTS5Query`. I confirmed the new tests fail against the old code.
- **Validator**: the rule-ID parsing moved into `configRuleID(msg)` with `len(msg) > 7`. New `ruleid_internal_test.go` includes the exact-7-character case.
- **Path checks**: `validatePathWithinRoot` and watcher `isInDir` now treat `..` only as a whole path element (`rel == ".."` or `"../"` prefix, plus `filepath.IsAbs`). The dropped `"/.."` check was dead code, since `filepath.Rel` never returns a leading separator. The tests were un-skipped and extended (`..cache`, `sub/..hidden` accepted; `..`, `../x`, `sub/../../x` rejected; `..foo.md` counts as inside the dir).
- **CLI**: decided that **both** `parse` and `build` reject a `-dir` that is missing or not a directory (exit 1, `workspace directory "…" does not exist` / `… is not a directory`), using a shared `requireDir` in `cmd/jokateko/main.go`. `build` no longer creates the missing dir or exports an empty board. The e2e static-export test runs `init` first, so it is unaffected. COMMANDS.md updated.
- Verification: `go test ./...` passes; `-count=3 -shuffle=on` passes; gofmt and vet are clean; `make cover` total is 93.2%. The only remaining skip is `TestGetVCSFallback`, which needs `-buildvcs`. Installed binary smoke test: `parse`/`build -dir /nonexistent` exit 1 and nothing is created; `parse -dir .` exits 0.
- Not done here (out of scope per the task): proxy handshake replay for go-sdk 1.8 `server/discover`; gentypes pointer and embedded-field handling.

## Completion Summary
- **Completed At:** 2026-10-05T06:17:58Z

### What Was Done
- formatFTS5Query passes a quoted query through only when it is a single well-formed phrase (non-blank, no embedded quotes); everything else is sanitized. TestSearchMalformedQuotes un-skipped, plus new table cases.
- validator: new configRuleID helper with a `len(msg) > 7` bounds check, and a test.
- config validatePathWithinRoot and watcher isInDir treat `..` only as a whole path element; skipped test re-enabled and extended.
- cmd/jokateko: a shared requireDir makes parse and build exit 1 when -dir is missing or not a directory; build no longer creates the directory. COMMANDS.md updated.

### Why / Rationale
Each fix is the smallest change that removes the failure at its source, with a test that fails on the old code. build fails on a missing -dir too, because silently exporting an empty board hides typos the same way the false OK from parse did. The proxy go-sdk handshake and gentypes pointer support are split into their own follow-up tasks.
