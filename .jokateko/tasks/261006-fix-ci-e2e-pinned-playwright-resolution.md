+++
title = 'Fix CI e2e: pinned Playwright resolution and repo-relative test paths'
status = 'in_review'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['ci', 'release']
summary = 'make e2e-test fails in CI before any test runs: bunx pulls a second Playwright copy, specs hardcode /workspaces/jokateko paths, and the binary is not built yet. Make the e2e run independent of local devcontainer state.'
created_at = '2026-10-06T13:50:25Z'
changed_at = '2026-10-06T13:52:53Z'
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
- [ ] CI run is green after push (human)

## Notes

### [2026-10-06 13:52 UTC]

Implemented. Makefile: `PLAYWRIGHT := web/node_modules/.bin/playwright`; a new `node_modules` target (`ln -sfn web/node_modules`); `e2e-test` now depends on `node_modules build`; `lighthouse-test` and `screenshots` depend on `node_modules`. Tests: new `tests/e2e/helpers/paths.ts` (`REPO_ROOT`, `WEB_DIR`, `BINARY_PATH` from `__dirname`), used in test-server.ts, static-export*.spec.ts and valibot-resilience.spec.ts. Workflows unchanged: the step order now works because `e2e-test` builds first.

Verified: after removing the root symlink, `make e2e-test` recreated it and 60/60 passed. A fresh `git worktree` in a scratch dir outside /workspaces also passed 60/60. `make lint test` is green. The CI run after push is still open (human).
