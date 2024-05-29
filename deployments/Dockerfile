FROM golang:1.21 as builder

WORKDIR /opt/app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o flowbot github.com/flowline-io/flowbot/cmd


FROM alpine:latest

RUN apk add --no-cache bash

WORKDIR /opt/app

# Fetch build from Github.
COPY --from=builder /opt/app/flowbot .
COPY deployments/entrypoint.sh .

RUN chmod +x flowbot && chmod +x entrypoint.sh
RUN mkdir subscribe

ENTRYPOINT ./entrypoint.sh

# HTTP
EXPOSE 6060
