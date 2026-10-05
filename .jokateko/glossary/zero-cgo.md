+++
title = 'Zero-CGO'
tags = ['backend']
summary = 'Building every Go binary with CGO_ENABLED=0 and only pure-Go dependencies, so Jokateko cross-compiles to one static executable per platform with no C toolchain or system libraries.'
+++

With `CGO_ENABLED=0`, Go cannot call C code. Every dependency must therefore be pure Go, which is why SQLite comes from `modernc.org/sqlite` (a transpiled, C-free SQLite) instead of `mattn/go-sqlite3`.

**Why:** one `make cross-compile` produces static binaries for linux/darwin/windows × amd64/arm64 from any host, the Docker image can be `FROM scratch`, and users install nothing besides the binary.

**Consequence:** the Go race detector (`go test -race`) needs CGO and a C compiler, so it cannot run in the default devcontainer.

See the strategy *Pure Go Zero-CGO & Minimal Dependencies*.
