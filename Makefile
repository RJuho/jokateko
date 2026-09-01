.PHONY: all build test clean generate ui-build lint

# Binary name and output directory
BINARY_NAME := jokateko
BIN_DIR := bin

# Version injection variables for link time
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X 'github.com/RJuho/jokateko/internal/version.Version=$(VERSION)' \
           -X 'github.com/RJuho/jokateko/internal/version.Commit=$(COMMIT)' \
           -X 'github.com/RJuho/jokateko/internal/version.Date=$(DATE)' \
           -s -w

all: test build

# Generate type-safe queries using sqlc (https://github.com/sqlc-dev/sqlc)
generate:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate

# Run all Go tests with CGO disabled
test:
	CGO_ENABLED=0 go test -v ./...

# Build the Go binary with injected link-time version flags
build:
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
