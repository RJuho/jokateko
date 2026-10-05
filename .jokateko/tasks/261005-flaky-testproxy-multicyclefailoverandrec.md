+++
title = 'Flaky TestProxy_MultiCycleFailoverAndReconnect close-count assertion'
status = 'backlog'
priority = 'medium'
tags = ['testing', 'proxy', 'flaky']
summary = 'TestProxy_MultiCycleFailoverAndReconnect sometimes fails with "cycle 3: expected at least 3 local engine closes, got: 2". The likely cause: switchToProxy sets proxy mode before it closes the old engine and increments the close counter, so the test can check the counter too early.'
created_at = '2026-10-05T14:12:47Z'
changed_at = '2026-10-05T14:12:47Z'
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
- [ ] Root cause confirmed, for example by adding a delay in `oldEngine.Close()` locally to reproduce it reliably, or with `go test -count=N` under CPU load (`-cpu`, parallel packages)
- [ ] Fixed at the right level: either the test waits for the counter (e.g. include `closes >= cycle` in the `waitFor` condition), or `switchToProxy` updates the counter so it is consistent with the published state. Record the choice and the reason in Notes
- [ ] Check that `switchToStandalone` and the `localStarts` / `proxyDisconnects` counters do not have the same ordering gap, and that the final `starts`/`closes` checks cannot race
- [ ] `go test -count=200 -run TestProxy_MultiCycleFailoverAndReconnect ./internal/proxy` passes, and a full `make test` passes
- [ ] `-race` run handed to the human (the devcontainer has no C compiler): `CGO_ENABLED=1 go test -count=200 -race -run TestProxy_MultiCycleFailoverAndReconnect ./internal/proxy`
