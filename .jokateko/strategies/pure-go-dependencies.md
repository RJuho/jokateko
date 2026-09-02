+++
title = "Pure Go Zero-CGO & Minimal Dependencies"
tier = 1
summary = "Strict CGO_ENABLED=0 requirement, locked minimal dependencies, and Go 1.27+ baseline"
tags = ["backend", "docs"]
+++

# Pure Go Zero-CGO & Minimal Dependencies

## 1. Zero CGO (`CGO_ENABLED=0`)
All backend code must compile with `CGO_ENABLED=0` to ensure auditable, frictionless cross-compilation across Linux, macOS, and Windows without external C toolchains. Pure Go drivers (such as `modernc.org/sqlite`) must always be used.

## 2. Compiler Toolchain
Development and production builds must target at least **Go 1.27** or newer.

## 3. Minimal Locked Dependencies
Do not introduce new external libraries or frameworks without explicit human approval. Rely on the Go standard library, standard TOML parser (`go-toml/v2`), pure Go SQLite (`modernc.org/sqlite`), `fsnotify`, `goldmark`, and the official Model Context Protocol Go SDK.
