+++
title = 'Project Structure'
tier = 2
tags = ['backend', 'docs', 'frontend']
summary = 'Map of the most important folders, config files and generated artifacts in the Jokateko repository, and the rules for where new code, tests and configuration belong.'
+++

# Project Structure

Jokateko ships as **one pure-Go binary** (CLI, daemon, MCP server) that embeds a **Bun-built Preact UI**. The repository's `.jokateko/` folder dogfoods the tool itself and holds all project documentation as strategies and glossary terms. Package-level detail: *Go packages and data flow* and *Web UI architecture*. The user-facing `.jokateko/` layout: *Workspace and entity file format*.

## Directory map

```mermaid
flowchart TB
    root(["jokateko/ · one pure-Go binary embedding the Bun-built UI"])

    subgraph GO ["Go backend"]
        direction TB
        cmdJ["cmd/jokateko/<br/>CLI: serve · mcp · build · parse · init · version · about · licenses"]
        cmdGen["cmd/gentypes · cmd/genlicenses<br/>code generators"]
        core["internal/ model · parser · validator<br/>domain types, Markdown + TOML, DAG checks"]
        data["internal/ store · watcher · writer<br/>in-memory SQLite (sqlc), fsnotify, atomic writes"]
        api["internal/ service · server · mcp · proxy<br/>shared mutations, REST + SSE, MCP tools, stdio proxy"]
        misc["internal/ config · csp · exporter · version<br/>TOML config, Content-Security-Policy, static export, licenses"]
    end

    subgraph WEB ["web/ · Bun + Preact UI"]
        direction TB
        src["src/<br/>components · state · schemas · utils · hooks"]
        scripts["scripts/<br/>bundle.ts · dev.ts · check-tailwind.ts"]
        dist["dist/ (generated)<br/>index.html.gz · mermaid.min.js.gz"]
        embed["embed.go<br/>go:embed dist"]
    end

    subgraph QA ["Tests"]
        direction TB
        e2e["tests/e2e/ · Playwright"]
        lh["tests/lighthouse/ · Lighthouse"]
        chaos["tests/chaos/ · Go chaos test"]
    end

    subgraph META ["Project & infra"]
        direction TB
        jk[".jokateko/<br/>tasks · milestones · strategies · glossary"]
        infra[".github/workflows · .devcontainer · Dockerfile"]
    end

    root --> GO
    root --> WEB
    root --> QA
    root --> META
    scripts -. builds .-> dist
    dist -. embedded .-> embed
```

## Generated artifacts and their sources

Never edit the right-hand side by hand. Change the source and regenerate.

```mermaid
flowchart LR
    pkg["web/package.json + bun.lock"] -->|bun run build| dist["web/dist/*<br/>index.html.gz, mermaid.min.js.gz,<br/>mermaid.json (version + SRI), hashes.json"]
    srcUI["web/src/**"] -->|bun run build| dist
    dist -->|go:embed| bin["bin/jokateko"]

    models["internal/model/*.go"] -->|go run ./cmd/gentypes| ts["web/src/types/generated.ts"]
    sql["internal/store/schema.sql<br/>queries.sql + sqlc.yaml"] -->|sqlc generate| sqlgo["internal/store/queries.sql.go<br/>models.go · querier.go"]
    gomod["go.mod (direct requires)"] -->|go run ./cmd/genlicenses| lic["internal/version/licenses.json<br/>web/src/data/licenses.json"]
    pkg -->|go run ./cmd/genlicenses| lic

    jkdir[".jokateko/**"] -->|jokateko build| export["dist-kanban/index.html<br/>static export (gitignored)"]
```

`make generate` runs sqlc, gentypes and genlicenses. `make build` rebuilds `web/dist` when UI sources change, then compiles the binary. Because `web/dist` is generated and gitignored, `go install …@latest` cannot build Jokateko. Use `make build`.

## Key configuration files

| File | Purpose |
|---|---|
| `go.mod` / `go.sum` | Go module and locked dependencies (no CGO, see *Pure Go Zero-CGO & Minimal Dependencies*) |
| `web/package.json` / `web/bun.lock` | UI dependencies. The lockfile pins exact versions and integrity hashes, including Mermaid's CDN version and SRI |
| `Makefile` | Single entry point: `build`, `install`, `test`, `cover`, `e2e-test`, `lighthouse-test`, `generate`, `fuzz-*`, `chaos-test`, `cross-compile` |
| `.jokateko/config.toml` | This repo's Jokateko project config (columns, tags, server, CSP). Validate with `jokateko parse` |
| `internal/config/default.toml` | Compiled-in defaults and the reference config spec |
| `internal/store/sqlc.yaml` | sqlc code generation config |
| `web/biome.json` · `web/tsconfig.json` · `web/bunfig.toml` | UI lint and format, TypeScript, Bun dev-server Tailwind plugin |
| `playwright.config.ts` · `playwright.lighthouse.config.ts` · `tsconfig.json` (root) | E2E and Lighthouse suites, and typing for `tests/` |
| `.mcp.json` · `.agents/mcp_config.json` · `.claude/settings.json` · `skills-lock.json` | AI agent tooling: MCP servers (Jokateko, Playwright, Valibot, gopls) and skills |
| `README.md` · `AGENTS.md` (`CLAUDE.md` is a symlink) | User guide with the CLI reference; agent workflow rules |
| `.github/workflows/*.yml` · `.devcontainer/*` · `Dockerfile` | CI, devcontainer image and release builds |

## Rules

1. **Go code** lives in `internal/<package>/` (not importable from outside). Entry points only go in `cmd/`. Tests sit next to the code as `*_test.go`.
2. **UI code** lives in `web/src/`:
   - components by view (`components/board|calendar|strategies|glossary|modal|common`)
   - signals state in `state/`
   - Valibot schemas in `schemas/`
   - pure helpers in `utils/`
   - unit tests as `*.test.ts` next to the code
3. **E2E tests** go in `tests/e2e/*.spec.ts`, using `data-testid` selectors. Spec files are parsed as plain JS by the Bun-backed runner (no type annotations).
4. **`.jokateko/`** is mutated only through the Jokateko MCP tools or the UI. Never edit it directly.
5. **Documentation** lives in `.jokateko/strategies` and `.jokateko/glossary`, not in a `docs/` folder. The only root Markdown files are the README, contributor and security docs, and the agent instructions.
6. **Generated and build output** (`web/dist/`, `bin/`, `dist-kanban/`, `test-results/`, `playwright-report/`, `lighthouse-report/`) is gitignored. Regenerate it, never commit it.
7. **New dependencies** in `go.mod` or `web/package.json` need explicit human approval.
