# flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/flowline-io/flowbot)](https://goreportcard.com/report/github.com/flowline-io/flowbot)

flowbot is system for chatbot

## Features

- Chat bot
- Message Publish/Subscribe Hub
- Message Cron, Trigger, Task, Pipeline
- Workflow Action
- LLM Agents

## Architecture

<img src="./docs/architecture.png" alt="Architecture" align="center" width="100%" />

## Requirements

This project requires Go 1.23 or newer

## Run

```shell
# copy config and setting
cp docs/config.yaml flowbot.yaml

# build
go build -v -o tmp github.com/flowline-io/flowbot/cmd

# run
chmod +x tmp
./tmp
```

# License

Assistant Bot is licensed under the https://github.com/flowline-io/flowbot#GPL-3.0-1-ov-file.
