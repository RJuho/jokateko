.PHONY: all build test clean generate install-tools ui-build lint e2e-test

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

all: test build

# Install development and code generation tools
install-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate type-safe queries using sqlc and TypeScript models
generate:
	sqlc generate -f internal/store/sqlc.yaml
	go run ./cmd/gentypes

# Run all Go tests with CGO disabled
test:
	CGO_ENABLED=0 go test -v ./...

# Build the Go binary with injected link-time version and CSP asset hash flags
build:
	@if [ ! -f web/dist/script.sha256 ]; then $(MAKE) ui-build; fi
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/jokateko

# Bundle Web UI using Bun
ui-build:
	cd web && bun run build

# Run code diagnostics and vet
lint:
	CGO_ENABLED=0 go vet ./...

# Clean built artifacts
clean:
	rm -rf $(BIN_DIR)

# Run Playwright end-to-end tests
e2e-test:
	bunx playwright test
