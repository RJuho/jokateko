+++
title = 'Harden CI and release workflows for a public repo'
status = 'backlog'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['ci', 'security', 'release']
summary = 'Pin GitHub Actions by commit SHA, apply least-privilege permissions, make CI work for fork PRs, and fix the release job that never publishes the Docker `latest` tag.'
created_at = '2026-10-05T10:58:16Z'
changed_at = '2026-10-05T10:58:16Z'
+++

## Context
`.github/workflows/{ci,devcontainer,release}.yml` were written for a private repo.
- Actions are pinned by tag (`actions/checkout@v7`, `softprops/action-gh-release@v3`, …), not by commit SHA. That conflicts with the project's pinned-asset security priority.
- CI and release run inside `ghcr.io/rjuho/jokateko-devcontainer:latest`. Fork PRs can pull that image only if the package is public.
- Release `docker` job: `type=raw,value=latest,enable=${{ github.ref_name == 'main' }}` is never true on a `v*.*.*` tag push, so `ghcr.io/rjuho/jokateko:latest` is never published. Meanwhile the devcontainer job pushes `latest` on every release.
- Workflow-level `permissions: contents: write, packages: write` applies to every release job.

## Acceptance Criteria
- [ ] Every third-party action pinned to a full commit SHA with a `# vX.Y.Z` comment (Dependabot keeps them current; see the Dependabot task)
- [ ] `permissions` scoped per job to the minimum each job needs; workflow default is `contents: read`
- [ ] Docker `latest` tag published on stable `vX.Y.Z` tags, not on pre-releases (`-rc`)
- [ ] Pre-release tags (`v*-rc*`) produce a GitHub pre-release
- [ ] CI runs green for a PR from a fork (public devcontainer image, or documented fallback)
- [ ] The CI web-change detection (`git diff HEAD~1`) is correct for multi-commit pushes, or replaced
- [ ] `checksums.txt` produced and documented; provenance attestation (`actions/attest-build-provenance`) evaluated. Adding it needs human approval
- [ ] Human checklist in Notes: make the `jokateko` and `jokateko-devcontainer` ghcr packages public and link them to the repo
