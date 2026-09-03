+++
title = 'Phase 11: End-to-End Validation & Automated Testing'
status = 'done'
priority = 'critical'
milestone = '260902-mvp'
tags = ['backend', 'frontend', 'devops', 'ci-cd']
summary = 'Playwright test suite, unified devcontainer with on-demand AI tooling (make install-ai-tools), minimal scratch CLI Docker container, and GitHub Actions CI/CD with @devcontainers/ci.'
dependencies = ['260902-phase-10-bun-web-ui']
+++

# Phase 11: End-to-End Validation & Automated Testing

Deterministic automated feedback loops, Playwright E2E testing, unified devcontainer architecture with on-demand AI agent tooling, minimal scratch CLI image, and GitHub Actions CI/CD.

## Acceptance Criteria
- [x] 11.1 Playwright E2E Test Suite Setup (Chromium headless, make e2e-test)
- [x] 11.2 E2E Test Scenarios (board, task mutation, static export, valibot resilience, markdown typography)
- [x] 11.3 Project Dogfooding (jokateko init, parse, MCP tools, build export)
- [x] 11.4 Unified Devcontainer & On-Demand AI Tooling (.devcontainer/Dockerfile with non-opinionated tools, and make install-ai-tools for gopls, caw, and antigravity)
- [x] 11.5 Minimal Scratch CLI Dockerfile (`Dockerfile` with multi-stage build, zero-CGO static binary, non-root user, ca-certs)
- [x] 11.6 Devcontainer CI & Codebase Validation Workflows (.github/workflows/devcontainer.yml triggered on .devcontainer/Dockerfile changes or workflow_dispatch, and .github/workflows/ci.yml running validation inside released container)
- [x] 11.7 Release Automation Workflow (`.github/workflows/release.yml` with multi-arch Docker image published to GHCR and cross-compiled static binaries for Linux, macOS, and Windows)

## Completion Summary
- **Completed At:** 2026-09-03T07:35:00Z

### What Was Done
Implemented Playwright E2E test suite covering live kanban operations, task mutations, static HTML export, Valibot schema resilience, and typography rendering. Created a single, unified `.devcontainer/Dockerfile` based on `oven/bun:slim` with core build utilities, Go 1.27, Playwright Chromium headless, and non-opinionated developer CLI tools (`tree`, `ripgrep`, `fd-find`, `jq`, `bat`). Decoupled rapidly-evolving AI agent tooling (`gopls` Go MCP server, `caw`, and Google `antigravity` CLI) into an on-demand `make install-ai-tools` Makefile target. Added root `Dockerfile` using `FROM scratch` with non-root user and SSL certs for minimal CLI container usage. Established `.github/workflows/devcontainer.yml` utilizing `@devcontainers/ci` to build and publish the devcontainer to GHCR whenever `.devcontainer/Dockerfile` changes or on `workflow_dispatch` (e.g. upstream base updates), `.github/workflows/ci.yml` running code validation directly inside the published container, and `.github/workflows/release.yml` for multi-arch GHCR publishing and cross-platform static binary compilation (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`).

### Why / Rationale
Completes the foundational MVP automated quality gate, guarantees reproducible development environments between local developers and CI, avoids stale baked AI binaries while respecting developer choice, and delivers zero-dependency, zero-CVE distribution options across bare-metal binaries and containerized CLI runners.
