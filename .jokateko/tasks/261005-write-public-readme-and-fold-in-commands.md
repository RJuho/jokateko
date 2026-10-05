+++
title = 'Write public README and fold in COMMANDS.md'
status = 'backlog'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release']
summary = 'Replace the 14-line README with a complete public landing page (what/why, install, quickstart, MCP setup, CLI reference, security note) and delete COMMANDS.md.'
dependencies = ['261005-migrate-docs-into-strategies-and-glossar']
created_at = '2026-10-05T10:58:27Z'
changed_at = '2026-10-05T10:58:27Z'
+++

## Context
`README.md` is 14 lines and links to `docs/`, which is being removed. The decision is to fold `COMMANDS.md` into the README. User-facing CLI goes here; developer `make` targets go to CONTRIBUTING.md.

Note: `go install github.com/RJuho/jokateko/cmd/jokateko@latest` does **not** work, because the embedded `web/dist` is generated and gitignored. Do not advertise it. Offer release binaries, Docker, or build from source.

## Acceptance Criteria
- [ ] Intro: what Jokateko is, who it is for (humans + AI agents), Tasks-as-Code / Spec-First in 2–3 sentences
- [ ] Feature list (live board + calendar, strategies/glossary, MCP server + stdio proxy, static single-file export, `parse` linting, zero-CGO single binary)
- [ ] Screenshot of the board, stored in the repo at a small size (e.g. `.github/assets/`)
- [ ] Install: release binaries with checksum verification, Docker `ghcr.io/rjuho/jokateko` (volume mount example), build from source (`make build`, prerequisites Go 1.27+ and Bun)
- [ ] Quickstart: `jokateko init`, `jokateko serve`, open the UI, create a first task
- [ ] Connect an AI agent: `.mcp.json` example for `jokateko mcp`, explaining proxy vs standalone mode
- [ ] Configuration: `.jokateko/config.toml` overlay model, link to `internal/config/default.toml`
- [ ] CLI reference from COMMANDS.md: serve, mcp, parse/lint, build (`--mermaidjs`), init (`--replace`), version, help. The "Suggested Additional Commands" wording becomes real commands
- [ ] Security note: binds to loopback by default and has no authentication; do not expose it publicly (link SECURITY.md)
- [ ] "Dogfooding" section: this repo's own board lives in `.jokateko/`; how to browse it (`jokateko serve` or `jokateko build`)
- [ ] Contributing + license sections linking CONTRIBUTING.md, SECURITY.md, LICENSE
- [ ] `COMMANDS.md` deleted; no remaining references (`git grep COMMANDS.md`), including in `AGENTS.md`
- [ ] Every command in the README is run once and verified to work as written
