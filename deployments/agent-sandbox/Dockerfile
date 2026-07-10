# flowbot-agent-sandbox: ephemeral Cloud Agent base image.
# Do not COPY project sources; the orchestrator mounts /workspace at runtime.

ARG GO_VERSION=1.26.3
ARG NODE_MAJOR=22

FROM ubuntu:24.04 AS base

ARG GO_VERSION
ARG NODE_MAJOR
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG VCS_REF=unknown

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

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
    && curl -fsSL https://deb.nodesource.com/setup_${NODE_MAJOR}.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && npm install -g oxfmt oxlint \
    && rm -rf /var/lib/apt/lists/*

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

WORKDIR /workspace

LABEL org.opencontainers.image.title="flowbot-agent-sandbox" \
      org.opencontainers.image.description="Ephemeral Cloud Agent sandbox with git and common language toolchains" \
      org.opencontainers.image.url="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.source="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.created=${BUILD_DATE} \
      org.opencontainers.image.revision=${VCS_REF} \
      org.opencontainers.image.licenses="GPL-3.0"

USER agent

FROM base AS playwright

USER root

RUN npx --yes playwright install-deps chromium

USER agent

RUN npx --yes playwright install chromium
