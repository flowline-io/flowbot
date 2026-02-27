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

ENTRYPOINT exec /opt/app/flowbot
