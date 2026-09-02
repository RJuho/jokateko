# Jokateko

Local, Markdown-driven Kanban and task management tool engineered for human developers and AI agents.

## Configuration

Jokateko embeds sensible defaults directly into the binary. To customize your board (columns, priorities, tags, translations, server port, etc.), add a `.jokateko/config.toml` (or `config.toml`) file to your workspace. Any specified fields cleanly overlay the defaults.

You can inspect the full default reference specification here:
- **[Default TOML Configuration](internal/config/default.toml)**

## Documentation

- [CLI Commands](COMMANDS.md)
- [Architecture & Specifications](docs/README.md)