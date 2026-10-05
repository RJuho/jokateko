# Changelog

All notable changes to Jokateko are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - TBD

First public release.

### Added

- **Tasks-as-Code workspace:** tasks, milestones, strategies and glossary terms as Markdown files with TOML frontmatter in `.jokateko/`, with `jokateko init` scaffolding and a commented `config.toml` that overlays built-in defaults.
- **Live Web UI** (`jokateko serve`): Kanban board with drag and drop, filters, search and per-column sorting; month and week calendar of due dates and milestone timeframes; strategies and glossary views; Mermaid diagrams; light and dark theme. Changes on disk appear live over Server-Sent Events.
- **Task workflow:** acceptance-criteria checklists, notes, dependencies with cycle detection, created/changed/target timestamps, locked specs outside editable columns, and `complete_task` with a recorded completion summary.
- **Built-in MCP server** with tools for every entity, resources and a `next_task` prompt. `jokateko mcp` proxies to a running `serve` for the same folder or runs standalone, and fails over live in both directions.
- **Configurable board:** columns with owners and agent instructions, priorities, strategy tiers, tag vocabulary, UI translations and project locale.
- **Static export** (`jokateko build`): the whole board as one self-contained HTML file, with Mermaid from a pinned CDN (SRI), bundled, or off.
- **Validation** (`jokateko parse`): config, frontmatter, references and dependency cycles, with rule IDs and exit codes for CI.
- **One binary:** pure Go without CGO and with the web UI embedded, for Linux and macOS (amd64/arm64) and Windows (amd64), plus a `FROM scratch` Docker image. `about` and `licenses` list third-party licenses.

### Security

- Binds to `127.0.0.1` by default; Cross-Origin protection, DNS-rebinding `Host` check, JSON-only API writes and a hash-based Content Security Policy. See [SECURITY.md](SECURITY.md).

[Unreleased]: https://github.com/RJuho/jokateko/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/RJuho/jokateko/releases/tag/v0.1.0
