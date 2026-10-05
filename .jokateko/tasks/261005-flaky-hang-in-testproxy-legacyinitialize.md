+++
title = 'Flaky hang in TestProxy_LegacyInitializeSurvivesSwitches'
status = 'done'
priority = 'high'
tags = ['backend']
summary = 'Fixed an intermittent hang in TestProxy_LegacyInitializeSurvivesSwitches. A failover requested while a proxy switch was still finishing timed out, and the proxy then silently dropped the client request. Switches are now serialized, undeliverable calls get a JSON-RPC error, and test reads have a deadline.'
created_at = '2026-10-05T07:23:39Z'
changed_at = '2026-10-05T10:31:31Z'
+++

## Context

This was found on 2026-10-05 while verifying the translation and locale work. It is **not** caused by that work: it reproduces on a clean worktree of `HEAD` (5470573).

- **Repro:** `go test -count=40 -timeout 90s -run TestProxy_LegacyInitializeSurvivesSwitches ./internal/proxy` hangs in roughly 1 of 3 batches of 40, on both clean HEAD and the working tree. One normal `go test ./...` run hit it and hung for 10 minutes.
- **Stack:** the test goroutine is blocked in `rawClient.expectResponse` (`internal/proxy/handshake_test.go:195`), called from the switch loop at `handshake_test.go:237/250`. A go-sdk `newIOConnLimited` reader is idle in `io.pipe.read`. So a response to the replayed legacy `initialize` after a switch is never delivered: either a lost or raced message, or an unbuffered pipe deadlock.
- **Likely related:** commit 2e4784f, "MCP proxy: replay the go-sdk 1.8 server/discover handshake on failover".

## Acceptance Criteria
- [x] Root cause of the missing response identified, whether it's a proxy bug or a test-harness race.
- [x] Fix applied. `expectResponse` gets a read deadline so a regression fails fast instead of hanging.
- [x] `go test -count=200 -race -run TestProxy_LegacyInitializeSurvivesSwitches ./internal/proxy` passes.

## Notes

### [2026-10-05 07:45 UTC]

**Root cause: proxy bug, not a test race.** Confirmed with a live goroutine dump and `go test -overlay` debug logging.

1. `switchToProxy` sets `isProxy=true` and then closes the old local engine. It clears `switching` only in its defer. `waitMode(true)` returns inside that window, so the test runs `call(3)`, `stop()` and sends request 4 while that switch is still flagged as in flight.
2. Request 4's Write to the dead daemon fails (logged: `EOF` on a stale keep-alive conn). The Client→Backend pump calls `switchToStandalone`, which saw `switching==true` and polled for `!switching && isStandalone`. The in-flight switch ends in *proxy* state, so the poll can never succeed, and after 500 ms it returned `timeout waiting for concurrent switch`.
3. The pump then dropped request 4 silently and the runner stayed in proxy mode on the dead daemon. The poll loop skips proxy mode, and ErrRejected write errors don't fail the streamable conn, so nothing recovered. The dump matched: no local-engine goroutines, and the reader parked in `streamableClientConn.Read`.

**Fix** (`internal/proxy/proxy.go`):
- New `switchMu` serializes `switchToProxy` and `switchToStandalone`. A switch requested during another now waits for it, then re-checks state and switches if still needed. The 50×10 ms polling loops and the "timeout waiting for concurrent switch" error are gone.
- New `Runner.forward` holds the write → failover → retry logic. A call that still can't be delivered gets a JSON-RPC `-32603` error response, so the client can't hang.

**Tests:**
- `rawClient.readLine` adds a 5 s deadline, used by `expectResponse` and `waitID`.
- `TestSwitchWaitsForInFlightSwitch` replaces `TestSwitchWaitsForConcurrentSwitch`, which encoded the old timeout behaviour.
- New `TestForward`.

**Verification:**
- `go test -count=200 -run TestProxy_LegacyInitializeSurvivesSwitches` passes.
- 40×40 extra iterations had no hang (before the fix it hung within 5–8 batches).
- `go test -count=20 ./internal/proxy`, `go test ./...` and `go vet ./...` are clean.
- **`-race` was not run:** it needs cgo and this container has no C compiler. Please run `CGO_ENABLED=1 go test -count=200 -race -run TestProxy_LegacyInitializeSurvivesSwitches ./internal/proxy` on a host with gcc.

### [2026-10-05 10:31 UTC]

Criterion 3 was ticked when the human approved completion. `-count=200` passed **without** `-race`; the race-detector run is still pending because the devcontainer has no C compiler.

## Completion Summary
- **Completed At:** 2026-10-05T10:31:31Z

### What Was Done
- internal/proxy/proxy.go: new Runner.switchMu serializes switchToProxy and switchToStandalone. Each holds it for the whole switch and re-checks the state after acquiring it. The 50x10ms polling loops and the "timeout waiting for concurrent switch" error are removed.
- internal/proxy/proxy.go: the Client->Backend pump now calls a new Runner.forward (write, fail over to standalone if in proxy mode, retry). A call that still can't be delivered gets a -32603 error response, so the client never waits forever.
- Tests: rawClient.readLine puts a 5 s deadline on reads and is used by expectResponse and waitID. TestSwitchWaitsForInFlightSwitch replaces TestSwitchWaitsForConcurrentSwitch, which encoded the old timeout behaviour. New TestForward covers delivery, failover and the error response.
- Verified: go test -count=200 on the test, a 40x40 repro loop with no hang (it hung within 5-8 batches before), go test ./... and go vet ./... clean. The -race run was not done in the devcontainer (no C compiler).

### Why / Rationale
The old wait loop treated "another switch is running" as "the other switch will produce my target state". That isn't true when the in-flight switch goes the other way. A mutex expresses the real requirement: one switch at a time, decided against the state left by the previous one. Lock order is switchMu then mu, and the pumps only take mu, so it can't deadlock. Replying with an error for undeliverable calls removes the whole silent-hang class, even if a failover fails for some other reason.
