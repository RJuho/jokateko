+++
title = 'Fix CI e2e: pinned Playwright resolution and repo-relative test paths'
status = 'done'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['ci', 'release']
summary = 'make e2e-test no longer depends on local devcontainer state. It uses the lockfile-pinned Playwright through a Makefile-managed root link, builds the binary first, and the tests resolve paths from the repo root. CI is green.'
created_at = '2026-10-06T13:50:25Z'
changed_at = '2026-10-06T14:02:11Z'
+++

## Problem
In CI (checkout at `/__w/jokateko/jokateko`) every spec fails with "Playwright Test did not expect test.describe() to be called here", followed by "No tests found". The causes:

1. The repo root `node_modules -> web/node_modules` symlink exists only in the local devcontainer (it's gitignored and was created by hand). Without it, `bunx playwright` downloads `playwright@1.63.0` into the global bun cache. The configs and specs then resolve `@playwright/test` through another module instance.
2. The specs and helper hardcode `/workspaces/jokateko/...` (binary, `web/dist/*`, `web/node_modules/mermaid`).
3. `ci.yml` runs `make e2e-test` before `make build`, but the tests start `bin/jokateko`.

## Acceptance Criteria
- [x] Makefile `node_modules` target creates the root symlink; the Playwright targets depend on it
- [x] Makefile runs the lockfile-pinned `web/node_modules/.bin/playwright` instead of `bunx playwright`
- [x] `e2e-test` depends on `build`
- [x] No absolute `/workspaces/jokateko` paths left in tests; a shared `tests/e2e/helpers/paths.ts` derives them from the repo root
- [x] `make e2e-test` passes after the root symlink is removed, and from a checkout outside `/workspaces/jokateko`
- [x] CI run is green after push (human)

## Notes

### [2026-10-06 13:52 UTC]

Implemented. Makefile: `PLAYWRIGHT := web/node_modules/.bin/playwright`; a new `node_modules` target (`ln -sfn web/node_modules`); `e2e-test` now depends on `node_modules build`; `lighthouse-test` and `screenshots` depend on `node_modules`. Tests: new `tests/e2e/helpers/paths.ts` (`REPO_ROOT`, `WEB_DIR`, `BINARY_PATH` from `__dirname`), used in test-server.ts, static-export*.spec.ts and valibot-resilience.spec.ts. Workflows unchanged: the step order now works because `e2e-test` builds first.

Verified: after removing the root symlink, `make e2e-test` recreated it and 60/60 passed. A fresh `git worktree` in a scratch dir outside /workspaces also passed 60/60. `make lint test` is green. The CI run after push is still open (human).

## Completion Summary
- **Completed At:** 2026-10-06T14:02:11Z

### What Was Done
- Makefile: `PLAYWRIGHT := web/node_modules/.bin/playwright` replaces `bunx playwright` in e2e-test, lighthouse-test and screenshots. A new `node_modules` target runs `ln -sfn web/node_modules node_modules`, and those targets depend on it. e2e-test also depends on `build`.
- New tests/e2e/helpers/paths.ts exports REPO_ROOT, WEB_DIR and BINARY_PATH, derived from `__dirname`. It replaces the hardcoded /workspaces/jokateko paths in test-server.ts, static-export.spec.ts, static-export-xss.spec.ts, static-export-mermaid.spec.ts and valibot-resilience.spec.ts.
- Verified: 60/60 without the root link, 60/60 from a worktree outside /workspaces, `make lint test` green, and the GitHub CI run green after pushing 6e4a4e3.

### Why / Rationale
The root configs and specs import @playwright/test from the repo root, so they need a root node_modules that points at the same pinned install the CLI uses. Otherwise bunx fetches a second copy and the module instances clash. Making the link a Makefile target keeps a single install in web/ and makes the setup reproducible. Calling the pinned CLI directly prevents silent version drift. Deriving paths from the file location makes the tests work in any checkout (CI uses /__w/...). Building inside e2e-test matches lighthouse-test and screenshots, and fixes the workflow step order without editing the workflows.
