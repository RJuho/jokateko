# Contributing

Thanks for helping! Jokateko is developed with Jokateko: the plan, the architecture rules and the vocabulary live in [`.jokateko/`](.jokateko/). Read the tier-1 [architecture](.jokateko/strategies/architecture.md) and [dependency](.jokateko/strategies/pure-go-dependencies.md) strategies before your first change.

## Setup

**Devcontainer (recommended).** Open the repository in VS Code (or any devcontainer tool) and choose *Reopen in Container*. The image (`ghcr.io/rjuho/jokateko-devcontainer`) includes Go, Bun and Playwright Chromium, and runs `make skills` on creation.

**Manual.** You need Go 1.27+, [Bun](https://bun.sh) 1.4+ and `make`. `make install-tools` installs `sqlc`, which only `make generate` needs. Run `make skills` once to restore the AI agent skills listed in `skills-lock.json`.

## Make targets

| Target | Does |
|---|---|
| `build` · `install` | Build `bin/jokateko` (rebuilds the web UI if needed); copy it to `INSTALL_DIR` (default `~/.local/bin`) |
| `test` · `cover` | Go tests with `CGO_ENABLED=0`; `cover` also writes `coverage.out` and prints total coverage |
| `lint` | `gofmt` check and `go vet` |
| `e2e-test` · `lighthouse-test` | Playwright end-to-end tests; Lighthouse audits of every view (reports in `lighthouse-report/`) |
| `generate` | Regenerate sqlc queries, TypeScript types and license data |
| `fuzz-markdown` · `fuzz-api` · `fuzz-mcp` · `fuzz-all` | Go fuzzing (`FUZZTIME=10s` by default) |
| `chaos-test` | Concurrent REST + MCP + file-edit stress test |
| `cross-compile` · `docker-build` | Release binaries for all platforms with `checksums.txt`; the `FROM scratch` image |
| `skills` · `skills-update` | Restore the agent skills; maintainers: update them and `skills-lock.json` |
| `clean` · `clean-cache` | Remove `bin/`; clear Go caches and test reports |

## Workflow

1. **Every change has a Jokateko task.** Create one in Backlog (UI or MCP) with a summary and `- [ ]` acceptance criteria, and include the task file in your pull request.
2. **Change `.jokateko/` only through the Web UI or the MCP tools**, never by editing the files. Agents get this rule automatically; see [AGENTS.md](AGENTS.md).
3. **Read the relevant strategies first**, and update them in the same PR when behaviour changes.
4. **No new dependencies** in `go.mod` or `web/package.json` without maintainer approval. Go code must build with `CGO_ENABLED=0`.
5. Move the task to **In Review** when you're done. A maintainer reviews it and completes it.

## Tests

- **Go:** standard `testing`, next to the code as `*_test.go`.
- **Web UI:** `bun test` unit tests next to the code, Playwright specs in `tests/e2e/`. Target elements only through `data-testid`.
- **Before a PR:** `make lint test e2e-test` and `jokateko parse` must pass.
- **Race detector:** `go test -race` needs CGO and a C compiler, which the devcontainer doesn't have. Run it on a host that does when you touch concurrency.

## AI-assisted contributions

Welcome, and held to the same review bar as any other change. The agent works through the Jokateko MCP tools and leaves the task In Review. A human reviews the result, makes the commit, and is responsible for it.

## Commits and pull requests

- One task per pull request. The PR description names the task ID (for example `261005-fix-login-redirect`).
- Commit subject: `Complete task: <task title>`, or a short imperative sentence for smaller changes.

## Releases (maintainers)

1. Make sure `main` is green and every task in the release milestone is done.
2. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`.
3. The release workflow cross-compiles the binaries, publishes them with `checksums.txt` and generated release notes, and pushes the Docker images to `ghcr.io/rjuho`. A tag with a suffix (`v0.1.0-rc1`) becomes a GitHub pre-release and doesn't move the `latest` image tag.
