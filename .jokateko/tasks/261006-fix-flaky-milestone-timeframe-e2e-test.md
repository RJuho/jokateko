+++
title = 'Fix flaky milestone timeframe e2e test'
status = 'done'
priority = 'medium'
tags = ['e2e', 'testing']
summary = 'The milestone timeframe e2e test was flaky because of two live-refresh bugs in the Web UI: milestone fields derived from tasks went stale, and changes made before the first SSE open were lost. Both are fixed in the frontend; the test passes 50/50 runs.'
created_at = '2026-10-06T08:51:24Z'
changed_at = '2026-10-06T13:41:53Z'
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
- [x] Root cause identified (watcher ordering, SSE timing or test sequencing)
- [x] Fix in the product if it is a real refresh bug, otherwise make the test wait on a deterministic signal
- [x] `--repeat-each=50` on the test passes with no failures

## Notes

### [2026-10-06 13:40 UTC]

Root cause: two real live-refresh bugs in the Web UI, not a test problem.

1. **Stale derived milestone fields.** The server derives `target_timeframe`, progress and `is_archived` from tasks when it reads them, but a task event broadcasts only the task. If `/api/milestones` was served before the watcher ingested the tasks, the card kept showing `2026-12-31` permanently. The baseline run without the fix reproduced this: 1/50 failed with `toContainText`.
2. **Gap between the initial fetch and the SSE subscription.** `App.tsx` fetches and opens `/api/events` in parallel. A broadcast sent after the fetch was served but before the subscriber was registered was lost (the "element not found" variant). `sse.ts` only resynced on *re*connects.

Fix (frontend only):
- `web/src/state/sse.ts`: resync board and entities on every `onopen`, including the first. The server registers the subscriber before the stream opens, so nothing is missed. The `needsResync` flag is removed.
- `web/src/state/sse.ts`: task created/updated/deleted events schedule a coalesced (100 ms) milestone refetch.
- `web/src/state/store.ts`: new `fetchLiveMilestones()`, which `fetchLiveEntities()` now reuses.
- New `web/src/state/sse.test.ts` covers the first-open resync and the coalescing.

Verification: `--repeat-each=50` gives 50/50 passed (baseline: 49/50). Full e2e suite 60/60, `bun test src/state` 25/25, typecheck, `go test ./...` all pass.

## Completion Summary
- **Completed At:** 2026-10-06T13:41:53Z

### What Was Done
- `web/src/state/sse.ts` resyncs the board and entities on every SSE `onopen`, including the first, and the `needsResync` flag is removed.
- Task created/updated/deleted events schedule a coalesced (100 ms) milestone refetch.
- `web/src/state/store.ts` has a new `fetchLiveMilestones()`, which `fetchLiveEntities()` reuses.
- New `web/src/state/sse.test.ts` covers the first-open resync and the coalescing.

### Why / Rationale
The server derives milestone timeframe, progress and archive state from tasks when it reads them, and task events carry only the task, so the client must refetch milestones after task changes. A coalesced client refetch also covers tasks moving between milestones and task deletes, with no server change. The SSE hub registers the subscriber before the stream opens, so a snapshot taken on open plus the events that follow is complete. That closes the gap between the initial fetch and the subscription.
