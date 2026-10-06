+++
title = 'Flaky TestProxy_MultiCycleFailoverAndReconnect close-count assertion'
status = 'done'
priority = 'medium'
tags = ['flaky', 'proxy', 'testing']
summary = 'Fixed the flaky close-count check in TestProxy_MultiCycleFailoverAndReconnect. switchToProxy now closes and counts the old local engine before it publishes proxy mode.'
created_at = '2026-10-05T14:12:47Z'
changed_at = '2026-10-06T08:45:25Z'
+++

## Context
Seen once during a full `make test` while doing task `261005-repository-cleanup-and-portable-agent-to` (no Go code had changed):

```
internal/proxy/proxy_test.go:406: cycle 3: expected at least 3 local engine closes, got: 2
```

It then passed 30/30 run alone, 10/10 at package level, and in two more full-suite runs. So it only shows up under parallel load.

**Likely cause (from reading the code, not yet confirmed):** `switchToProxy` (`internal/proxy/proxy.go` ~605–622) sets `isProxy = true` and `localEngine = nil` under `r.mu`. It then releases the lock, and only after that calls `oldEngine.Close()` and `r.localCloses.Add(1)`. The test's `waitFor` checks `IsProxyMode() && !LocalEngineRunning()`, which becomes true as soon as the lock is released. Then the test calls `listTasks` and reads `Stats()`. If `Close()` is slow (SQLite store, watcher), the counter has not been incremented yet. So this is a race in the test's assertion, not a failed failover.

Related: `261005-flaky-hang-in-testproxy-legacyinitialize` (another proxy timing flake, done).

## Acceptance Criteria
- [x] Root cause confirmed, for example by adding a delay in `oldEngine.Close()` locally to reproduce it reliably, or with `go test -count=N` under CPU load (`-cpu`, parallel packages)
- [x] Fixed at the right level: either the test waits for the counter (e.g. include `closes >= cycle` in the `waitFor` condition), or `switchToProxy` updates the counter so it is consistent with the published state. Record the choice and the reason in Notes
- [x] Check that `switchToStandalone` and the `localStarts` / `proxyDisconnects` counters do not have the same ordering gap, and that the final `starts`/`closes` checks cannot race
- [x] `go test -count=200 -run TestProxy_MultiCycleFailoverAndReconnect ./internal/proxy` passes, and a full `make test` passes
- [x] `-race` run handed to the human (the devcontainer has no C compiler): `CGO_ENABLED=1 go test -count=200 -race -run TestProxy_MultiCycleFailoverAndReconnect ./internal/proxy`

## Notes

### [2026-10-06 07:35 UTC]

**Root cause confirmed.** `switchToProxy` published `isProxy=true` / `localEngine=nil` and released `r.mu` before it called `oldEngine.Close()` and `localCloses.Add(1)`. The test's `waitFor` and its `list_tasks` round trip could finish inside that gap.
- Repro with a temporary `time.Sleep(100ms)` at the top of `localEngine.Close()`: failed 10/10 at cycle 1 ("got: 0").
- Repro without the delay, under load (16 busy-loop processes on 8 CPUs, `-count=150 -cpu=1,4` = 300 runs): 38/300 failed. Failures came at cycle 1 (18), cycle 2 (8) and cycle 3 (12).

**Two fixes tried against the delayed repro:**
- A (test only): add `closes >= cycle` to the per-cycle `waitFor`. Proxy package 10x: green, 24.5s.
- B (production): in `switchToProxy`, close the old engine and increment `localCloses` inside the `r.mu` section that publishes proxy mode. Proxy package 10x: green, 24.5s, no hangs. `TestSwitchWaitsForInFlightSwitch` passed.

**Chose B.** It makes the published state honest for every reader: "proxy mode, no local engine" now means the engine is closed. A only covers this one test. Cost: forward/pump `RLock` callers wait for the duration of `Close()` during a switch. `Close()` never takes `r.mu`, so there is no deadlock risk, and the delayed run showed no timing regression. A was not kept, because with B it would be redundant. Rejected: incrementing the counter under the lock but before `Close()`, because the counter would then mean "retired", not "closed".

**Counter audit:**
- `localStarts`, `proxyConnects` and `proxyDisconnects` were already incremented in the same locked section that publishes the state.
- `switchToStandalone` closes only `oldConn` after unlocking, and that close has no counter.
- The final `starts`/`closes` checks follow the per-cycle check and a standalone `waitFor`, so they cannot race.
- `coverage_test.go:262` reads connects/disconnects, which are set under the lock: safe.

**Validation (delay removed):**
- `-count=200 -run TestProxy_MultiCycleFailoverAndReconnect`: ok.
- Same load test as the repro: 0/300 failures.
- `go test -count=20 ./internal/proxy`: ok.
- `make test`: ok.

`-race` is still for the human to run (no C compiler in the devcontainer).

### [2026-10-06 08:45 UTC]

**`-race` run deferred.** The human has no Go toolchain on the host and will run it later: `CGO_ENABLED=1 go test -count=200 -race -run TestProxy_MultiCycleFailoverAndReconnect ./internal/proxy`. The task is completed with this criterion open.

## Completion Summary
- **Completed At:** 2026-10-06T08:45:25Z

### What Was Done
In `internal/proxy/proxy.go` `switchToProxy`, `oldEngine.Close()` and `localCloses.Add(1)` now run inside the `r.mu` section that sets `isProxy`/`localEngine`, not after the unlock. Confirmed with an injected 100 ms delay in `Close()` (fails 10/10 without the fix) and under CPU load (38/300 failures before, 0/300 after). `-count=200`, proxy package `-count=20` and `make test` all pass. The `-race` run has been handed to the human, who will run it later (no Go on the host).

### Why / Rationale
The fix is in production code, not only in the test (option A was a waitFor on the counter), so every reader of the runner state sees a consistent view: proxy mode means the engine is already closed. Holding the lock during `Close()` is safe because `Close()` never takes `r.mu`, and the delayed runs showed no slowdown. The other counters were already updated together with the state they describe.
