+++
title = 'v0.1.0 release dry-run and make repository public'
status = 'backlog'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['release']
summary = 'Human-driven final gate: cut a release candidate, verify binaries, Docker image and README instructions end to end, then tag v0.1.0 and flip the repository to public.'
dependencies = ['261005-dependabot-config-and-changelog-for-v010', '261005-harden-ci-and-release-workflows-for-a-pu', '261005-pre-publication-audit-history-secrets-li', '261005-write-contributingmd-and-securitymd', '261005-write-public-readme-and-fold-in-commands', '261005-stop-raw-html-in-markdown-from-running-s']
created_at = '2026-10-05T10:59:07Z'
changed_at = '2026-10-05T12:18:45Z'
+++

## Context
Last step of the milestone, performed by the maintainer. Agents can prepare and verify, but tagging, publishing, and changing repository visibility are human actions.

## Acceptance Criteria
- [ ] Tag `v0.1.0-rc1`; release workflow green; GitHub pre-release has all 5 binaries + `checksums.txt`
- [ ] Binaries smoke-tested on Linux, macOS (arm64), and Windows: `version`, `init`, `serve`, `mcp`, `build`
- [ ] Docker image `ghcr.io/rjuho/jokateko:<rc>` runs `serve` with a mounted workspace
- [ ] README install, quickstart, and MCP steps followed verbatim on a clean machine
- [ ] ghcr packages (`jokateko`, `jokateko-devcontainer`) public and linked to the repo
- [ ] Repo settings: description, topics, website; private vulnerability reporting, secret scanning, and push protection enabled; branch protection on `main` requiring CI; fork PR workflows require approval for first-time contributors
- [ ] Repository visibility switched to public
- [ ] CI runs green for a test pull request from a fork (deferred from `261005-harden-ci-and-release-workflows-for-a-pu`; needs the public devcontainer image)
- [ ] Tag `v0.1.0`; CHANGELOG release date set; `latest` Docker tag published
- [ ] Milestone `Public Release v0.1.0` closed
