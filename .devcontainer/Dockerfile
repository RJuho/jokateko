FROM oven/bun:slim

# Use apt-get instead of apk, and update package names
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    bash \
    coreutils \
    make \
    curl \
    procps \
    unzip \
    # For AI use
    tree \
    ripgrep \
    fd-find \
    jq \
    bat \
    sqlite3 \
    # Clean up apt cache to keep image size small
    && rm -rf /var/lib/apt/lists/* \
    && sed -i 's|/bin/sh|/bin/bash|' /etc/passwd

RUN ln -s $(which fdfind) /usr/local/bin/fd \
    && ln -s $(which batcat) /usr/local/bin/bat

RUN bunx playwright install-deps chromium

USER bun

ENV PATH="/home/bun/.local/bin/go/bin:/home/bun/go/bin:/home/bun/.local/bin:${PATH}"

RUN mkdir -p /home/bun/.local/bin \
    && curl -L https://github.com/04mg/caw/releases/latest/download/caw-linux-amd64 \
    -o /home/bun/.local/bin/caw \
    && chmod +x /home/bun/.local/bin/caw

RUN mkdir -p /home/bun/.local/bin \
    && curl -L https://go.dev/dl/go1.27.0.linux-amd64.tar.gz \
    -o /home/bun/.local/bin/go.tar.gz \
    && tar -C /home/bun/.local/bin -xzf /home/bun/.local/bin/go.tar.gz \
    && rm /home/bun/.local/bin/go.tar.gz

RUN echo 'export PS1="\e[01;32m\u\e[m:\e[01;34m\w\e[m\$ "' >> /home/bun/.bashrc

RUN curl -fsSL https://antigravity.google/cli/install.sh | bash

RUN bunx playwright install chromium