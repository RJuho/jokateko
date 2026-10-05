+++
title = 'Write CONTRIBUTING.md and SECURITY.md'
status = 'backlog'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'security', 'release']
summary = 'Document how to contribute (devcontainer, make targets, tests, dependency rule, Tasks-as-Code workflow, AI-assisted PRs) and how to report vulnerabilities, including the local-tool threat model.'
dependencies = ['261005-write-public-readme-and-fold-in-commands', '261005-repository-cleanup-and-portable-agent-to']
created_at = '2026-10-05T10:58:52Z'
changed_at = '2026-10-05T10:58:52Z'
+++

## Context
External contributors need the rules that so far lived only in AGENTS.md and in people's heads. The server has no authentication. Its protection is loopback binding, Cross-Origin protection, a DNS-rebinding Host check, and CSP/SRI. The threat model must say so plainly.

## Acceptance Criteria
### CONTRIBUTING.md
- [ ] Dev setup: devcontainer (recommended; image on ghcr) and manual setup (Go 1.27+, Bun, sqlc via `make install-tools`), plus `make skills` for agent skills
- [ ] Make targets moved from COMMANDS.md: build, install, test, cover, e2e-test, lighthouse-test, generate, fuzz-*, chaos-test, cross-compile, docker-build
- [ ] Testing expectations: Go tests next to the code, Playwright with `data-testid`, `jokateko parse` must pass. Race detector runs need a C compiler (not in the devcontainer)
- [ ] Strict dependency policy (zero CGO, no new libraries without maintainer approval), linking the `pure-go-dependencies` strategy
- [ ] Tasks-as-Code workflow: each change references a Jokateko task. `.jokateko/` is changed only through the MCP tools or the UI. Read the relevant strategies first
- [ ] AI-assisted contributions welcome; same review bar; humans own the commit
- [ ] Commit/PR conventions and the release process (from the CHANGELOG task)
### SECURITY.md
- [ ] Supported versions (latest minor)
- [ ] Private reporting via GitHub Security Advisories; expected response time
- [ ] Threat model: single-user local tool; binds `127.0.0.1` by default; no auth, so never expose it on a network; `0.0.0.0` only in trusted containers; CORS and Cross-Origin protection; DNS-rebinding Host check; CSP hashes; Mermaid SRI pinning; static exports contain the full project data
- [ ] README links both files
