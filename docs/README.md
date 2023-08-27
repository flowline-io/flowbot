# extra chatbot framework

## ENV

```shell
CHANNEL_PATH=/subscribe
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=123456
TINODE_URL=http://127.0.0.1:6060
DOWNLOAD_PATH=/download
```

## extra json config

> See extra.conf

## Dev tools

```shell

# Generator cli
go run github.com/sysatom/flowbot/internal/cmd/composer generator bot -name example -rule input,group,agent,command,condition,cron,form
go run github.com/sysatom/flowbot/internal/cmd/composer generator vendor -name example

# Migrate cli
go run github.com/sysatom/flowbot/internal/cmd/composer migrate import

# Migration file cli
go run github.com/sysatom/flowbot/internal/cmd/composer migrate migration -name file_name
```

## Lint

```shell
# install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# check
golangci-lint run --timeout=10m --config=./server/extra/.golangci.yaml ./server/extra/...
```