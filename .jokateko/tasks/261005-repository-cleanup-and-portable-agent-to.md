+++
title = 'Repository cleanup and portable agent tooling'
status = 'in_review'
priority = 'medium'
milestone = '261005-public-release-v010'
tags = ['housekeeping', 'release']
summary = 'Tidy the repository root and make the AI-agent tooling work in any fresh clone: fix dangling skill symlinks, generalize AGENTS.md, and justify or remove each root-level file.'
dependencies = ['261005-migrate-docs-into-strategies-and-glossar']
created_at = '2026-10-05T10:58:36Z'
changed_at = '2026-10-05T11:53:45Z'
+++

## Context
Decision: keep the agent tooling (`.mcp.json`, `.agents/mcp_config.json`, `.claude/settings.json`, `skills-lock.json`, `AGENTS.md`/`CLAUDE.md`) as a dogfooding showcase, but make it portable.
- `.claude/skills/{daisyui,playwright-cli,use-modern-go,valibot}` are tracked symlinks into `.agents/skills/`, which is gitignored. They dangle in every fresh clone.
- `AGENTS.md` hardcodes `/home/bun/.local/bin/` and references `/docs/README.md` and `COMMANDS.md`.
- `.jokateko/config.toml` sets `host = "0.0.0.0"` (for devcontainer port forwarding), while the compiled default is `127.0.0.1`. Users copying this repo's config get a server on all interfaces.
- `Makefile install-ai-tools` pipes remote scripts into `bash` (Claude Code and Antigravity installers).

## Acceptance Criteria
- [x] A `make skills` target (or equivalent) restores `.agents/skills/` from `skills-lock.json` using the existing `bunx skills` tooling, so the `.claude/skills` symlinks resolve. Documented in CONTRIBUTING
- [x] `AGENTS.md` generalized: no `/home/bun` paths (use `make install` / `INSTALL_DIR`), points to Jokateko strategies instead of `docs/`, no `COMMANDS.md` reference, workflow rules still correct (In Review state, no self-commit)
- [x] `.jokateko/config.toml` uses `127.0.0.1`, or keeps `0.0.0.0` with a comment explaining it is for the devcontainer only. CORS origins are consistent with the choice
- [x] Each root-level file reviewed and kept, moved or removed, with the reason recorded in Notes: `tools.go`, root `tsconfig.json`, `playwright*.config.ts`, `skills-lock.json`, `.dockerignore`, `Dockerfile`
- [x] `install-ai-tools` reviewed: clearly labelled optional / dev-only, or removed
- [x] `.gitignore` covers all local artifacts (`.DS_Store`, reports, `coverage.out`, `.vscode` if unused). `git status` is clean after a full `make test e2e-test build`
- [x] Fresh-clone check: `git clone` into a temp dir, then `make build` and `make test` succeed without the devcontainer's pre-existing state

## Notes

### [2026-10-05 11:24 UTC]

Decision (2026-10-05, maintainer): keep `host = "0.0.0.0"` in this repo's `.jokateko/config.toml`. It is needed so the host can reach `jokateko serve` inside the devcontainer. Add a comment in the config explaining this. SECURITY.md should still say that the Host/DNS-rebinding check only covers loopback connections, so `0.0.0.0` exposes the unauthenticated API to the network. Use it only in a trusted container setup.

### [2026-10-05 11:53 UTC]

**Done**
- **Skills:** `make skills` runs `bunx skills@1.7.0 experimental_install` (CLI version pinned in `SKILLS_CLI`) and restores `.agents/skills/` from `skills-lock.json`. In a fresh clone the `.claude/skills/*` symlinks dangled before and resolve after, with a clean `git status`. `make skills-update` (maintainers) takes over the `bunx skills update -p -y` that used to live in `install-ai-tools`. The devcontainer runs `make skills` as `postCreateCommand`. The AGENTS.md hint is updated. CONTRIBUTING.md documentation is covered by task `261005-write-contributingmd-and-securitymd`, whose criteria already include `make skills`.
- **AGENTS.md:** no `/home/bun` (uses `make install` / `INSTALL_DIR`, default `~/.local/bin`), "In Review (`in_review`)" instead of "In Preview", "git commit" typo fixed, `web/dist/index.html` path, tool list completed (`list_task_items`, `set_task_target`). No docs/ or COMMANDS.md references.
- **`.jokateko/config.toml`:** kept `host = "0.0.0.0"` (maintainer decision) with a comment explaining it is for devcontainer access and warning that there is no auth. The maintainer moved the port to 8080, and CORS origins match. Edited directly because no MCP tool covers config.toml. `.devcontainer/devcontainer.json` now forwards and labels 8080 (was 3000).
- **install-ai-tools:** kept, and labelled as optional and not needed to build or test. The comment says it installs third-party CLIs at their latest versions by piping vendor install scripts to bash.

**Root-file review**
| File | Decision | Reason |
|---|---|---|
| `tools.go` | **removed** | It blank-imported only runtime libraries that `internal/` already imports directly. `go mod tidy` leaves `go.mod`/`go.sum` unchanged without it, and nothing references it |
| root `tsconfig.json` | keep | Types `tests/**` and `playwright.config.ts` for the editor and `tsc`; `web/tsconfig.json` only covers `web/` |
| `playwright.config.ts`, `playwright.lighthouse.config.ts` | keep at root | The suites live in root `tests/` and run via `web` scripts with `--config ../…`. Moving them would need path rewrites for no gain |
| `skills-lock.json` | keep | Source for `make skills`; the `skills` CLI expects it at the project root |
| `.dockerignore` | keep, extended | Added `lighthouse-report` and `coverage.out` to keep the build context small |
| `Dockerfile` | keep | Builds the release `FROM scratch` image (release workflow) |

**.gitignore:** already covers every artifact. After `make test`, `make e2e-test` and `make build`, `git status` showed only the intended edits. `.DS_Store` is ignored; `.vscode/` is an empty local directory, which git never tracks.

**Verification:** `make test` OK, `make e2e-test` 55/55, `make build` OK. Fresh clone (HEAD + this diff): `make skills`, `make build`, `make test` all OK.

**Flaky test seen (not caused by this change; no Go code changed):** `TestProxy_MultiCycleFailoverAndReconnect` (`internal/proxy/proxy_test.go:406`, "cycle 3: expected at least 3 local engine closes, got: 2") failed once in the first full `make test`. It then passed 30/30 alone, 10/10 at package level, and in two full-suite runs. It looks timing-dependent under parallel load, like the earlier `TestProxy_LegacyInitializeSurvivesSwitches` hang. Candidate follow-up task.
