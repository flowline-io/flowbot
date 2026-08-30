# flowbot-agent-sandbox: ephemeral Cloud Agent base image.
# Do not COPY project sources; the orchestrator mounts /workspace at runtime.
# The flowbot CLI is not baked in; chatagent sandbox injects flowbot-cli_linux_amd64 at runtime.

ARG GO_VERSION=1.26.6
ARG NODE_MAJOR=22
ARG DCG_VERSION=v0.6.7

# Download Destructive Command Guard (dcg) for host/sandbox PATH parity.
# Flowbot's chat-agent gate runs on the host; the image binary is for operational parity only.
FROM ubuntu:24.04 AS dcg-builder

ARG DCG_VERSION=v0.6.7

ENV DEBIAN_FRONTEND=noninteractive

COPY scripts/install-dcg.sh /tmp/install-dcg.sh
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl xz-utils \
    && rm -rf /var/lib/apt/lists/* \
    && DCG_VERSION="${DCG_VERSION}" sh /tmp/install-dcg.sh /out

FROM ubuntu:24.04 AS base

ARG GO_VERSION
ARG NODE_MAJOR
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG VCS_REF=unknown

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    GOROOT=/usr/local/go \
    GOTOOLCHAIN=local

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        build-essential \
        ca-certificates \
        coreutils \
        curl \
        findutils \
        gawk \
        git \
        grep \
        jq \
        openssh-client \
        python3 \
        python3-pip \
        python3-venv \
        ripgrep \
        sed \
        sudo \
        tzdata \
        wget \
    && ln -sf /usr/bin/python3 /usr/local/bin/python \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
        | tar -C /usr/local -xz \
    && rm -rf \
        /usr/local/go/test \
        /usr/local/go/doc \
        /usr/local/go/api \
        /usr/local/go/lib/wasm \
    && find /usr/local/go -type d -name testdata -prune -exec rm -rf {} + \
    && curl -fsSL https://deb.nodesource.com/setup_${NODE_MAJOR}.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && npm install -g oxfmt oxlint \
    && npm cache clean --force \
    && rm -rf \
        /var/lib/apt/lists/* \
        /var/cache/apt/archives/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/info \
        /root/.npm \
    && find /tmp -mindepth 1 -delete

RUN set -eux; \
    if getent group 1000 >/dev/null; then \
      group="$(getent group 1000 | cut -d: -f1)"; \
      if [ "$group" != "agent" ]; then groupmod -n agent "$group"; fi; \
    else \
      groupadd --gid 1000 agent; \
    fi; \
    if getent passwd agent >/dev/null; then \
      :; \
    elif getent passwd 1000 >/dev/null; then \
      user="$(getent passwd 1000 | cut -d: -f1)"; \
      usermod -l agent -d /home/agent -m "$user"; \
      usermod -g agent agent; \
    else \
      useradd --uid 1000 --gid agent --create-home --shell /bin/bash agent; \
    fi; \
    echo 'agent ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/agent; \
    chmod 0440 /etc/sudoers.d/agent

COPY --from=dcg-builder /out/dcg /usr/local/bin/dcg
COPY pkg/agent/dcg/config.toml /etc/dcg/config.toml

WORKDIR /opt/app

LABEL org.opencontainers.image.title="flowbot-agent-sandbox" \
      org.opencontainers.image.description="Ephemeral Cloud Agent sandbox with git and common language toolchains" \
      org.opencontainers.image.url="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.source="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.created=${BUILD_DATE} \
      org.opencontainers.image.revision=${VCS_REF} \
      org.opencontainers.image.licenses="GPL-3.0"

USER agent

# FaaS runs `go run main.go` with Network=none and GOPROXY=off (stdlib only).
# GOTOOLCHAIN=local keeps the bundled compiler from attempting a toolchain download.
RUN mkdir -p /tmp/faas-go-smoke \
    && printf '%s\n' 'module flowbotfn' '' 'go 1.26' > /tmp/faas-go-smoke/go.mod \
    && printf '%s\n' 'package main' '' 'import "fmt"' '' 'func main() { fmt.Println("{}") }' > /tmp/faas-go-smoke/main.go \
    && cd /tmp/faas-go-smoke \
    && GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go run main.go \
    && rm -rf /home/agent/.cache/go-build \
    && find /tmp -mindepth 1 -delete

FROM base AS playwright

USER root

# install-deps re-runs apt; strip docs in this layer.
RUN npx --yes playwright install-deps chromium \
    && npm cache clean --force \
    && rm -rf \
        /var/lib/apt/lists/* \
        /var/cache/apt/archives/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/info \
        /root/.npm \
    && find /tmp -mindepth 1 -delete

USER agent

# Keep the Playwright browser cache; drop npx cache only.
RUN npx --yes playwright install chromium \
    && npm cache clean --force \
    && rm -rf /home/agent/.npm \
    && find /tmp -mindepth 1 -delete
