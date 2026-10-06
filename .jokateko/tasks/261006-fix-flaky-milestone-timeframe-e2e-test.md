+++
title = 'Fix flaky milestone timeframe e2e test'
status = 'backlog'
priority = 'medium'
tags = ['testing', 'e2e']
summary = 'tests/e2e/timestamps.spec.ts "derives milestone target timeframe from tasks with target dates" fails about 1 in 10 runs: the milestone-timeframe element never appears after files are written to disk.'
created_at = '2026-10-06T08:51:24Z'
changed_at = '2026-10-06T08:51:24Z'
+++

## Context
Seen on 2026-10-06 while verifying `261006-use-the-new-logo-on-the-readme-cover-and`. The failure is unrelated to that change. `bunx playwright test tests/e2e/timestamps.spec.ts:134 --repeat-each=10` gives 1 failed / 9 passed:

```
Error: expect(locator).toBeVisible() failed
Locator: locator('[data-testid="milestone-timeframe-261001-q4-release"]')
Expected: visible
Error: element(s) not found
```

The test writes a milestone and two tasks straight to `server.dir/.jokateko/` with `writeFileSync`, then expects the UI to show the derived timeframe. This is likely a race between the fsnotify watcher / SSE refresh and the page load or navigation (for example, the milestone file is picked up before or after the tasks, or the page loads before the watcher broadcasts).

## Acceptance Criteria
- [ ] Root cause identified (watcher ordering, SSE timing or test sequencing)
- [ ] Fix in the product if it is a real refresh bug, otherwise make the test wait on a deterministic signal
- [ ] `--repeat-each=50` on the test passes with no failures
