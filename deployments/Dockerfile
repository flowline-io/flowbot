FROM golang:1.26 as builder

WORKDIR /opt/app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flowbot -ldflags "-X github.com/flowline-io/flowbot/version.Buildstamp=`date -u '+%Y-%m-%dT%H:%M:%SZ'` -X github.com/flowline-io/flowbot/version.Buildtags=`git describe --tags --always`" github.com/flowline-io/flowbot/cmd


FROM alpine:latest

RUN apk add --no-cache bash curl tzdata

WORKDIR /opt/app

COPY --from=builder /opt/app/flowbot .

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
