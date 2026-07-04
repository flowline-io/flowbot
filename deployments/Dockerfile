FROM golang:1.26 as builder

WORKDIR /opt/app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flowbot -ldflags "-X github.com/flowline-io/flowbot/version.Buildstamp=`date -u '+%Y-%m-%dT%H:%M:%SZ'` -X github.com/flowline-io/flowbot/version.Buildtags=`git describe --tags --always`" github.com/flowline-io/flowbot/cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o composer -ldflags "-X github.com/flowline-io/flowbot/version.Buildstamp=`date -u '+%Y-%m-%dT%H:%M:%SZ'` -X github.com/flowline-io/flowbot/version.Buildtags=`git describe --tags --always`" github.com/flowline-io/flowbot/cmd/composer


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

RUN mkdir -p /opt/app/chat-workspace
RUN chown -R 1000:1000 /opt/app/chat-workspace
RUN chmod -R 775 /opt/app/chat-workspace
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

ENTRYPOINT exec /opt/app/flowbot
