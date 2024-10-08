# extra chatbot framework

## Setup

### install dev tools
```shell
./docs/install.sh
```

## extra json config

> See extra.conf

## Dev tools

```shell

# Generator cli
go run github.com/flowline-io/flowbot/cmd/composer generator bot -name example -rule agent,command,cron,form,input,instruct
go run github.com/flowline-io/flowbot/cmd/composer generator vendor -name example

# Migrate cli
go run github.com/flowline-io/flowbot/cmd/composer migrate import

# Migration file cli
go run github.com/flowline-io/flowbot/cmd/composer migrate migration -name file_name
```

## Lint

```shell
# install
go install github.com/mgechev/revive@latest

# check
revive -formatter friendly ./...
```

## cloc

```shell
# install
sudo apt install cloc

# count
cloc --exclude-dir=node_modules --exclude-ext=json .
```

## security

```shell
go install golang.org/x/vuln/cmd/govulncheck@latest

# check
govulncheck ./...
```

## swagger

> https://github.com/swaggo/swag/blob/master/README.md

```shell
# install
go install github.com/swaggo/swag/cmd/swag@latest

# generate
swag init -g cmd/main.go

# format
swag fmt -g cmd/main.go
```
