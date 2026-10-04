# Base Jokateko devcontainer image served from GitHub Container Registry
ARG DEVCONTAINER_IMAGE=ghcr.io/rjuho/jokateko-devcontainer:latest

# ==============================================================================
# Builder stage: compiles Web UI and static binary using Jokateko devcontainer.
# Runs natively on the build host and cross-compiles for the target platform.
# ==============================================================================
FROM --platform=$BUILDPLATFORM ${DEVCONTAINER_IMAGE} AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Prepare unprivileged user and group for minimal scratch image
USER root
RUN echo "jokateko:x:10001:10001:Jokateko User:/workspace:/bin/false" > /etc/passwd.scratch \
    && echo "jokateko:x:10001:" > /etc/group.scratch

# Copy source repository
COPY --chown=bun:bun . .

USER bun

# Build self-contained Web UI and compile zero-CGO static executable
RUN make ui-build && GOOS=$TARGETOS GOARCH=$TARGETARCH make build

# ==============================================================================
# Final stage: minimal, zero-attack-surface container (FROM scratch)
# ==============================================================================
FROM scratch

# SSL certificates for secure outbound network requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Unprivileged user and group definitions
COPY --from=builder /etc/passwd.scratch /etc/passwd
COPY --from=builder /etc/group.scratch /etc/group

# Self-contained static Jokateko binary with embedded Web UI assets
COPY --from=builder /app/bin/jokateko /jokateko

# Run as non-root user
USER 10001:10001

# Default working directory for mounting local repositories
WORKDIR /workspace

# Web UI and API port, use default :8080 port
EXPOSE 8080

# Executable entrypoint: arguments passed to `docker run` are handled by jokateko
ENTRYPOINT ["/jokateko"]

# Default command starts the daemon server
CMD ["serve"]
