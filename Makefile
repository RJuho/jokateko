.PHONY: all build test clean generate install-tools install-ai-tools ui-build lint e2e-test cross-compile docker-build devcontainer-build

# Binary name and output directory
BINARY_NAME := jokateko
BIN_DIR := bin

# Version and CSP asset hash injection variables for link time
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
SCRIPT_HASH  = $(shell [ -f web/dist/script.sha256 ] && cat web/dist/script.sha256 2>/dev/null)
STYLE_HASH   = $(shell [ -f web/dist/style.sha256 ] && cat web/dist/style.sha256 2>/dev/null)

LDFLAGS = -X 'github.com/RJuho/jokateko/internal/version.Version=$(VERSION)' \
          -X 'github.com/RJuho/jokateko/internal/version.Commit=$(COMMIT)' \
          -X 'github.com/RJuho/jokateko/internal/version.Date=$(DATE)' \
          -X 'github.com/RJuho/jokateko/internal/version.ScriptHash=$(SCRIPT_HASH)' \
          -X 'github.com/RJuho/jokateko/internal/version.StyleHash=$(STYLE_HASH)' \
          -s -w

# Route Go compiler temp directories into workspace filesystem
GOTMPDIR ?= $(CURDIR)/.gopath/tmp
export GOTMPDIR
_mkdir := $(shell mkdir -p $(GOTMPDIR))

all: test build

# Install development and code generation tools
install-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install latest AI agent tooling (gopls Go MCP server, Caw CLI, and Google Antigravity CLI)
install-ai-tools:
	go install golang.org/x/tools/gopls@latest
	@mkdir -p /home/bun/.local/bin
	curl -L https://github.com/04mg/caw/releases/latest/download/caw-linux-amd64 -o /home/bun/.local/bin/caw && chmod +x /home/bun/.local/bin/caw
	curl -fsSL https://antigravity.google/cli/install.sh | bash

# Generate type-safe queries using sqlc, TypeScript models, and open-source licenses
generate:
	sqlc generate -f internal/store/sqlc.yaml
	go run ./cmd/gentypes
	go run ./cmd/genlicenses

# Run all Go tests with CGO disabled
test:
	@if [ ! -f internal/version/licenses.json ]; then go run ./cmd/genlicenses; fi
	@if [ ! -f web/dist/script.sha256 ]; then $(MAKE) ui-build; fi
	CGO_ENABLED=0 go test -v ./...

# Build the Go binary with injected link-time version and CSP asset hash flags
build:
	@if [ ! -f internal/version/licenses.json ]; then go run ./cmd/genlicenses; fi
	@if [ ! -f web/dist/script.sha256 ]; then $(MAKE) ui-build; fi
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/jokateko

# Bundle Web UI using Bun
ui-build:
	@if [ ! -d web/node_modules ]; then cd web && bun install --frozen-lockfile; fi
	cd web && bun run build

# Run code diagnostics and vet
lint:
	CGO_ENABLED=0 go vet ./...

# Clean built artifacts
clean:
	rm -rf $(BIN_DIR)

# Run Playwright end-to-end tests
e2e-test:
	@if [ ! -d web/node_modules ]; then cd web && bun install --frozen-lockfile; fi
	@mkdir -p $(GOTMPDIR)
	TMPDIR=$(GOTMPDIR) bunx playwright test

# Cross-compile static zero-CGO binaries across Linux, macOS, and Windows
cross-compile:
	@if [ ! -f web/dist/script.sha256 ]; then $(MAKE) ui-build; fi
	mkdir -p $(GOTMPDIR) $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/jokateko
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/jokateko
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/jokateko
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/jokateko
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/jokateko
	cd $(BIN_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	rm -rf $(GOTMPDIR)

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
	CGO_ENABLED=0 go test -v ./test/chaos -count=1
