+++
title = 'Dependabot config and CHANGELOG for v0.1.0'
status = 'backlog'
priority = 'medium'
milestone = '261005-public-release-v010'
tags = ['ci', 'dependencies', 'release']
summary = 'Add Dependabot for Go modules, Bun/npm, GitHub Actions and the devcontainer Dockerfile, and start a Keep-a-Changelog CHANGELOG.md seeded with v0.1.0.'
dependencies = ['261005-harden-ci-and-release-workflows-for-a-pu']
created_at = '2026-10-05T10:58:42Z'
changed_at = '2026-10-05T10:58:42Z'
+++

## Context
After the actions are SHA-pinned (CI hardening task), Dependabot keeps them and the locked dependencies current. Dependabot only proposes PRs. Each dependency PR is still reviewed by a human against the strict-dependency policy, and Dependabot never adds new dependencies.

## Acceptance Criteria
- [ ] `.github/dependabot.yml` with ecosystems: `gomod` (`/`), `npm` (`/web`; confirm Dependabot handles `bun.lock`, otherwise document the manual update flow), `github-actions` (`/`), `docker` (`/.devcontainer`, `/`)
- [ ] Weekly schedule, grouped minor/patch updates per ecosystem, sensible open-PR limit
- [ ] Verified that a Dependabot bump of `mermaid` in `web/bun.lock` still regenerates the pinned CDN version + SRI (`web/dist/mermaid.json`) via the normal build
- [ ] `CHANGELOG.md` in Keep a Changelog format with an `[Unreleased]` section and `v0.1.0` summarizing the shipped features (from the done tasks / MVP milestone)
- [ ] Release process documented in CONTRIBUTING: update CHANGELOG, tag, and how GitHub's generated release notes relate to it
