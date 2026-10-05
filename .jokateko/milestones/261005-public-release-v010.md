+++
title = 'Public Release v0.1.0'
status = 'open'
tags = ['release', 'docs']
summary = 'Prepare the repository for public use: move all project knowledge into Jokateko, add a real README and contributor/security docs, tidy the repo, harden CI, audit and ship v0.1.0.'
+++

## Goals
- Jokateko is the single home for project knowledge: `docs/` is migrated into Strategies and Glossary, then deleted.
- A newcomer can understand, install, and run Jokateko from `README.md` alone.
- The repo is clean and portable: no devcontainer-only paths, no dangling symlinks, every root file has a reason to exist.
- Contributors and security researchers know the rules (`CONTRIBUTING.md`, `SECURITY.md`).
- CI and release workflows work for a public repo and fork PRs, with actions pinned by SHA.
- History, dependencies and licenses are audited before visibility flips.

## Decisions (2026-10-05)
- Keep the git history as-is, but audit it for secrets first. No squash, no author rewrite.
- COMMANDS.md is folded into README; dev `make` targets move to CONTRIBUTING.
- AI-agent tooling (`.mcp.json`, `AGENTS.md`, skills) stays as a dogfooding showcase, made portable.
- In scope: CONTRIBUTING, SECURITY, Dependabot, CHANGELOG. Out of scope: Code of Conduct, issue/PR templates.
