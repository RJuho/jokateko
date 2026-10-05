+++
title = 'Flaky hang in TestProxy_LegacyInitializeSurvivesSwitches'
status = 'backlog'
priority = 'high'
tags = ['backend']
summary = 'internal/proxy TestProxy_LegacyInitializeSurvivesSwitches intermittently blocks forever in rawClient.expectResponse, so go test ./... hangs until the 10-minute timeout. It reproduces on clean HEAD 5470573.'
created_at = '2026-10-05T07:23:39Z'
changed_at = '2026-10-05T07:23:39Z'
+++

## Context

This was found on 2026-10-05 while verifying the translation and locale work. It is **not** caused by that work: it reproduces on a clean worktree of `HEAD` (5470573).

- **Repro:** `go test -count=40 -timeout 90s -run TestProxy_LegacyInitializeSurvivesSwitches ./internal/proxy` hangs in roughly 1 of 3 batches of 40, on both clean HEAD and the working tree. One normal `go test ./...` run hit it and hung for 10 minutes.
- **Stack:** the test goroutine is blocked in `rawClient.expectResponse` (`internal/proxy/handshake_test.go:195`), called from the switch loop at `handshake_test.go:237/250`. A go-sdk `newIOConnLimited` reader is idle in `io.pipe.read`. So a response to the replayed legacy `initialize` after a switch is never delivered: either a lost or raced message, or an unbuffered pipe deadlock.
- **Likely related:** commit 2e4784f, "MCP proxy: replay the go-sdk 1.8 server/discover handshake on failover".

## Acceptance Criteria
- [ ] Root cause of the missing response identified, whether it's a proxy bug or a test-harness race.
- [ ] Fix applied. `expectResponse` gets a read deadline so a regression fails fast instead of hanging.
- [ ] `go test -count=200 -race -run TestProxy_LegacyInitializeSurvivesSwitches ./internal/proxy` passes.
