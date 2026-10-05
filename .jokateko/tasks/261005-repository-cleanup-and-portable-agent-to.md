+++
title = 'Repository cleanup and portable agent tooling'
status = 'backlog'
priority = 'medium'
milestone = '261005-public-release-v010'
tags = ['housekeeping', 'release']
summary = 'Tidy the repository root and make the AI-agent tooling work in any fresh clone: fix dangling skill symlinks, generalize AGENTS.md, and justify or remove each root-level file.'
dependencies = ['261005-migrate-docs-into-strategies-and-glossar']
created_at = '2026-10-05T10:58:36Z'
changed_at = '2026-10-05T10:58:36Z'
+++

## Context
Decision: keep the agent tooling (`.mcp.json`, `.agents/mcp_config.json`, `.claude/settings.json`, `skills-lock.json`, `AGENTS.md`/`CLAUDE.md`) as a dogfooding showcase, but make it portable.
- `.claude/skills/{daisyui,playwright-cli,use-modern-go,valibot}` are tracked symlinks into `.agents/skills/`, which is gitignored. They dangle in every fresh clone.
- `AGENTS.md` hardcodes `/home/bun/.local/bin/` and references `/docs/README.md` and `COMMANDS.md`.
- `.jokateko/config.toml` sets `host = "0.0.0.0"` (for devcontainer port forwarding), while the compiled default is `127.0.0.1`. Users copying this repo's config get a server on all interfaces.
- `Makefile install-ai-tools` pipes remote scripts into `bash` (Claude Code and Antigravity installers).

## Acceptance Criteria
- [ ] A `make skills` target (or equivalent) restores `.agents/skills/` from `skills-lock.json` using the existing `bunx skills` tooling, so the `.claude/skills` symlinks resolve. Documented in CONTRIBUTING
- [ ] `AGENTS.md` generalized: no `/home/bun` paths (use `make install` / `INSTALL_DIR`), points to Jokateko strategies instead of `docs/`, no `COMMANDS.md` reference, workflow rules still correct (In Review state, no self-commit)
- [ ] `.jokateko/config.toml` uses `127.0.0.1`, or keeps `0.0.0.0` with a comment explaining it is for the devcontainer only. CORS origins are consistent with the choice
- [ ] Each root-level file reviewed and kept, moved or removed, with the reason recorded in Notes: `tools.go`, root `tsconfig.json`, `playwright*.config.ts`, `skills-lock.json`, `.dockerignore`, `Dockerfile`
- [ ] `install-ai-tools` reviewed: clearly labelled optional / dev-only, or removed
- [ ] `.gitignore` covers all local artifacts (`.DS_Store`, reports, `coverage.out`, `.vscode` if unused). `git status` is clean after a full `make test e2e-test build`
- [ ] Fresh-clone check: `git clone` into a temp dir, then `make build` and `make test` succeed without the devcontainer's pre-existing state
