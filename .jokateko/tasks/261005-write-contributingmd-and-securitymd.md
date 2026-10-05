+++
title = 'Write CONTRIBUTING.md and SECURITY.md'
status = 'done'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release', 'security']
summary = 'CONTRIBUTING.md and SECURITY.md are written. They cover the devcontainer and manual setup, make targets (including `make skills` and `make screenshots`), tests, the dependency rule, the Tasks-as-Code workflow, AI-assisted PRs, the release process, vulnerability reporting and the local-tool threat model.'
dependencies = ['261005-repository-cleanup-and-portable-agent-to', '261005-write-public-readme-and-fold-in-commands']
created_at = '2026-10-05T10:58:52Z'
changed_at = '2026-10-05T14:30:59Z'
+++

## Context
External contributors need the rules that so far lived only in AGENTS.md and in people's heads. The server has no authentication. Its protection is loopback binding, Cross-Origin protection, a DNS-rebinding Host check, and CSP/SRI. The threat model must say so plainly.

## Acceptance Criteria
### CONTRIBUTING.md
- [x] Dev setup: devcontainer (recommended; image on ghcr) and manual setup (Go 1.27+, Bun, sqlc via `make install-tools`), plus `make skills` for agent skills
- [x] Make targets moved from COMMANDS.md: build, install, test, cover, e2e-test, lighthouse-test, generate, fuzz-*, chaos-test, cross-compile, docker-build
- [x] Testing expectations: Go tests next to the code, Playwright with `data-testid`, `jokateko parse` must pass. Race detector runs need a C compiler (not in the devcontainer)
- [x] Strict dependency policy (zero CGO, no new libraries without maintainer approval), linking the `pure-go-dependencies` strategy
- [x] Tasks-as-Code workflow: each change references a Jokateko task. `.jokateko/` is changed only through the MCP tools or the UI. Read the relevant strategies first
- [x] AI-assisted contributions welcome; same review bar; humans own the commit
- [x] Commit/PR conventions and the release process (from the CHANGELOG task)
### SECURITY.md
- [x] Supported versions (latest minor)
- [x] Private reporting via GitHub Security Advisories; expected response time
- [x] Threat model: single-user local tool; binds `127.0.0.1` by default; no auth, so never expose it on a network; `0.0.0.0` only in trusted containers; CORS and Cross-Origin protection; DNS-rebinding Host check; CSP hashes; Mermaid SRI pinning; static exports contain the full project data
- [x] README links both files

## Notes

### [2026-10-05 12:00 UTC]

Added `CONTRIBUTING.md` and `SECURITY.md` in the README's minimal style. The README already linked both, and those links now resolve.

**Maintainer decisions (2026-10-05)**
- Security response time: best effort, no fixed number.
- Commit/PR convention: task-based. One task per PR, the PR names the task ID, and the commit subject is `Complete task: <title>` or a short imperative sentence.

**Release process:** CONTRIBUTING documents tag → release workflow (binaries, `checksums.txt`, generated notes, ghcr images). The "update CHANGELOG.md" step belongs to `261005-dependabot-config-and-changelog-for-v010`, which creates the file and already has a criterion to document the release process in CONTRIBUTING.

**Fix found while documenting:** `make lint` ran `gofmt -l … test` on the old directory. The chaos test moved to `tests/`, so it was never format-checked, and gofmt printed an lstat error. Changed it to `tests`; `make lint` passes.

**Verification:** every claim checked against the code: Makefile targets, the chaos test channels (file edits, REST, MCP), `bun test` (89 pass), `[mcp] allow_mutations`, the security middleware, and Mermaid SRI. `make lint` OK, `jokateko parse` OK.

## Completion Summary
- **Completed At:** 2026-10-05T14:30:59Z

### What Was Done
Added CONTRIBUTING.md (setup, make targets table, tests, dependency policy, task-based commit/PR convention, release process) and SECURITY.md (private reporting, best-effort response, threat model: loopback binding, Cross-Origin protection, a Host check that covers loopback connections only, CSP/SRI, no authentication). While documenting, fixed `make lint` to format-check `tests/` instead of the old directory. Every claim was checked against the code; `make lint` and `jokateko parse` pass. The `make screenshots` row was added with the README cover work. The task was waiting only on its dependency, the README task, which is now done.

### Why / Rationale
Public contributors need one place for the workflow and the dependency rules. Security reporters need a private channel and an honest threat model for an unauthenticated local tool. Checking each documented claim against the code keeps the docs from drifting from the Makefile and middleware on day one.
