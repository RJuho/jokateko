FROM oven/bun:slim

# Core build utilities and non-opinionated developer CLI tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    bash \
    coreutils \
    make \
    curl \
    procps \
    unzip \
    tree \
    ripgrep \
    fd-find \
    jq \
    bat \
    && rm -rf /var/lib/apt/lists/* \
    && sed -i 's|/bin/sh|/bin/bash|' /etc/passwd \
    && ln -s $(which fdfind) /usr/local/bin/fd \
    && ln -s $(which batcat) /usr/local/bin/bat \
    && git config --system --add safe.directory '*'

# Install Playwright browser dependencies (Chromium headless)
RUN bunx playwright install-deps chromium

USER bun

ENV PATH="/home/bun/.local/bin/go/bin:/home/bun/go/bin:/home/bun/.local/bin:${PATH}"

# Install pure Go compiler matching the image architecture (amd64 / arm64)
ARG GO_VERSION=1.27.1
ARG TARGETARCH
RUN mkdir -p /home/bun/.local/bin \
    && curl -fL https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz \
    -o /home/bun/.local/bin/go.tar.gz \
    && tar -C /home/bun/.local/bin -xzf /home/bun/.local/bin/go.tar.gz \
    && rm /home/bun/.local/bin/go.tar.gz

# Install Playwright Chromium headless browser binary
RUN bunx playwright install chromium

# Terminal prompt styling
RUN echo 'export PS1="\e[01;32m\u\e[m:\e[01;34m\w\e[m\$ "' >> /home/bun/.bashrc