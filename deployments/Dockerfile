FROM golang:1.26 AS builder

WORKDIR /opt/app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flowbot -ldflags "-X github.com/flowline-io/flowbot/version.Buildstamp=`date -u '+%Y-%m-%dT%H:%M:%SZ'` -X github.com/flowline-io/flowbot/version.Buildtags=`git describe --tags --always`" github.com/flowline-io/flowbot/cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o composer -ldflags "-X github.com/flowline-io/flowbot/version.Buildstamp=`date -u '+%Y-%m-%dT%H:%M:%SZ'` -X github.com/flowline-io/flowbot/version.Buildtags=`git describe --tags --always`" github.com/flowline-io/flowbot/cmd/composer

# Destructive Command Guard (dcg) for Always-on chat-agent run_terminal / run_code checks.
FROM alpine:latest AS dcg-builder

ARG DCG_VERSION=v0.6.7

COPY scripts/install-dcg.sh /tmp/install-dcg.sh
RUN apk add --no-cache ca-certificates curl tar xz \
    && DCG_VERSION="${DCG_VERSION}" sh /tmp/install-dcg.sh /out

FROM alpine:latest

RUN apk add --no-cache \
        bash curl wget tzdata ca-certificates \
        python3 py3-pip \
        git jq ripgrep \
        grep sed gawk findutils coreutils \
        openssh-client \
    && ln -sf /usr/bin/python3 /usr/local/bin/python \
    && update-ca-certificates

WORKDIR /opt/app

COPY --from=builder /opt/app/flowbot .
COPY --from=builder /opt/app/composer /usr/local/bin/composer
COPY --from=builder /opt/app/public ./public
COPY --from=dcg-builder /out/dcg /usr/local/bin/dcg
COPY pkg/agent/dcg/config.toml /etc/dcg/config.toml

RUN chmod +x flowbot

ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG VCS_REF=unknown

LABEL org.opencontainers.image.title="flowbot" \
      org.opencontainers.image.description="Homelab Data Hub & Capability Orchestration Center" \
      org.opencontainers.image.url="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.source="https://github.com/flowline-io/flowbot" \
      org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.created=${BUILD_DATE} \
      org.opencontainers.image.revision=${VCS_REF} \
      org.opencontainers.image.licenses="GPL-3.0"

HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \
  CMD wget -qO- http://127.0.0.1:6060/livez || exit 1

ENTRYPOINT exec /opt/app/flowbot
