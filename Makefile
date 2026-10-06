.PHONY: all build install test typecheck cover clean clean-cache generate install-tools install-ai-tools skills skills-update ui-build lint e2e-test lighthouse-test screenshots cross-compile docker-build devcontainer-build fuzz-markdown fuzz-api fuzz-mcp fuzz-all chaos-test

# Binary name, output directory and install location
BINARY_NAME := jokateko
BIN_DIR     := bin
INSTALL_DIR ?= $(HOME)/.local/bin

# Release targets for cross-compilation (GOOS/GOARCH)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Version injection variables for link time
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# CSP script/style hashes are derived at runtime from the embedded index.html,
# so they are intentionally not injected here (single source of truth).
LDFLAGS = -X 'github.com/RJuho/jokateko/internal/version.Version=$(VERSION)' \
          -X 'github.com/RJuho/jokateko/internal/version.Commit=$(COMMIT)' \
          -X 'github.com/RJuho/jokateko/internal/version.Date=$(DATE)' \
          -s -w

GO_BUILD = CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"

# Web UI bundle: rebuilt whenever any frontend source or dependency changes
UI_SRC := $(shell find web/src web/scripts -type f 2>/dev/null) web/index.html web/package.json web/bun.lock
UI_OUT := web/dist/index.html.gz

# Generated, git-tracked license report; regenerated only if missing (use `make generate` to refresh)
LICENSES := internal/version/licenses.json

# Lockfile-pinned Playwright CLI (bunx could fetch a second copy that clashes with @playwright/test)
PLAYWRIGHT := web/node_modules/.bin/playwright

all: test build

# Install development and code generation tools
install-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Pinned version of the agent-skills CLI (https://github.com/vercel-labs/skills)
SKILLS_CLI := skills@1.7.0

# Restore the project's agent skills into .agents/skills/ (gitignored) from skills-lock.json,
# so the .claude/skills/* symlinks resolve. Run once after cloning.
skills:
	bunx $(SKILLS_CLI) experimental_install

# Maintainers only: update all agent skills to their latest versions and rewrite skills-lock.json
skills-update:
	bunx $(SKILLS_CLI) update -p -y

# Optional, not needed to build or test: installs third-party AI agent CLIs (gopls MCP server,
# Claude Code, Google Antigravity) at their latest versions by piping each vendor's install script to bash
install-ai-tools:
	go install golang.org/x/tools/gopls@latest
	@mkdir -p $(INSTALL_DIR)
	curl -fsSL https://antigravity.google/cli/install.sh | bash
	curl -fsSL https://claude.ai/install.sh | bash

# Generate type-safe queries using sqlc, TypeScript models, and open-source licenses
generate:
	sqlc generate -f internal/store/sqlc.yaml
	go run ./cmd/gentypes
	go run ./cmd/genlicenses

web/node_modules: web/package.json web/bun.lock
	cd web && bun install --frozen-lockfile
	@touch $@

# Root link so the repo-root Playwright configs and tests resolve @playwright/test from web/
node_modules: web/node_modules
	ln -sfn web/node_modules $@

$(UI_OUT): $(UI_SRC) web/node_modules
	cd web && bun run build

$(LICENSES):
	go run ./cmd/genlicenses

# Bundle Web UI using Bun (only when sources changed)
ui-build: $(UI_OUT)

# Type-check the Web UI, including the drift check between the Valibot schemas and the generated Go types
typecheck: web/node_modules
	cd web && bun run typecheck

# Type-check the Web UI, then run all Go tests with CGO disabled
test: typecheck $(UI_OUT) $(LICENSES)
	CGO_ENABLED=0 go test -v ./...

# Run cmd/ and internal/ Go tests with a coverage profile (coverage.out) and print the total statement coverage
cover: $(UI_OUT) $(LICENSES)
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./cmd/... ./internal/... && go tool cover -func=coverage.out | tail -1

# Build the Go binary with injected link-time version flags (honours GOOS/GOARCH from the environment)
build: $(UI_OUT) $(LICENSES)
	@mkdir -p $(BIN_DIR)
	$(GO_BUILD) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/jokateko

# Build and install the binary into INSTALL_DIR (default ~/.local/bin)
install: build
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

# Run code diagnostics and vet
lint:
	@test -z "$$(gofmt -l cmd internal web tests)" || { echo "gofmt needed:"; gofmt -l cmd internal web tests; exit 1; }
	CGO_ENABLED=0 go vet ./...

# Clean built artifacts
clean:
	rm -rf $(BIN_DIR)

# Reclaim disk space: Go build/test/fuzz caches and Playwright outputs
clean-cache:
	go clean -cache -testcache -fuzzcache
	rm -rf test-results playwright-report web/test-results web/playwright-report lighthouse-report screenshots

# Run Playwright end-to-end tests against a fresh build (ensures the browser matching the pinned Playwright version; no-op if present)
e2e-test: node_modules build
	$(PLAYWRIGHT) install chromium
	$(PLAYWRIGHT) test

# Run Lighthouse audits of every Web UI view over Playwright Chromium (CDP); reports go to lighthouse-report/
lighthouse-test: node_modules build
	$(PLAYWRIGHT) install chromium
	$(PLAYWRIGHT) test --config playwright.lighthouse.config.ts

# Capture README screenshots of a demo workspace and render the cover; output goes to screenshots/
screenshots: node_modules build
	$(PLAYWRIGHT) install chromium
	$(PLAYWRIGHT) test --config playwright.screenshots.config.ts

# Cross-compile static zero-CGO binaries across Linux, macOS, and Windows
cross-compile: $(UI_OUT) $(LICENSES)
	@mkdir -p $(BIN_DIR)
	@set -e; for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ "$$os" = windows ] && ext=.exe; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GO_BUILD) -o $(BIN_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext ./cmd/jokateko; \
	done
	cd $(BIN_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt

# Build devcontainer image locally
devcontainer-build:
	docker build -t jokateko-devcontainer:latest -f .devcontainer/Dockerfile .

# Build production minimal scratch Docker container image using devcontainer
docker-build:
	docker build --build-arg DEVCONTAINER_IMAGE=jokateko-devcontainer:latest -t $(BINARY_NAME):latest .

FUZZTIME ?= 10s

# Run Fuzz testing for Markdown parser
fuzz-markdown:
	@echo "Fuzzing Markdown Parser..."
	CGO_ENABLED=0 go test -v ./internal/parser -fuzz=Fuzz -fuzztime=$(FUZZTIME)

# Run Fuzz testing for REST API
fuzz-api:
	@echo "Fuzzing REST API..."
	CGO_ENABLED=0 go test -v ./internal/server -fuzz=Fuzz -fuzztime=$(FUZZTIME)

# Run Fuzz testing for MCP Endpoint
fuzz-mcp:
	@echo "Fuzzing MCP Server..."
	CGO_ENABLED=0 go test -v ./internal/mcp -fuzz=Fuzz -fuzztime=$(FUZZTIME)

# Run all fuzz tests
fuzz-all: fuzz-markdown fuzz-api fuzz-mcp

# Run Multichannel Integration Chaos Test
chaos-test:
	@echo "Running Multichannel Chaos Test (10s)..."
	CGO_ENABLED=0 go test -v ./tests/chaos -count=1
