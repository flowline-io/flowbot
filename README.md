# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/flowline-io/flowbot)](https://goreportcard.com/report/github.com/flowline-io/flowbot)

Flowbot is a powerful chatbot system that provides message processing, workflow automation, and LLM agent capabilities.

## Key Features

- 🤖 Intelligent Chatbot
- 📨 Message Publish/Subscribe Hub
- ⏰ Message Cron, Trigger, Task, Pipeline
- 🔄 Configurable Workflow Actions
- 🧠 LLM Agent System

## Architecture

<img src="./docs/architecture.png" alt="Architecture" align="center" width="100%" />

## Getting Started

### Requirements

- Go 1.23 or higher
- OS: Linux/macOS/Windows

### Installation & Running

```shell
# 1. Clone the repository
git clone https://github.com/flowline-io/flowbot.git
cd flowbot

# 2. Configure
cp docs/config.yaml flowbot.yaml
# Modify flowbot.yaml as needed

# 3. Build
go build -v -o flowbot github.com/flowline-io/flowbot/cmd

# 4. Run
chmod +x flowbot
./flowbot
```

## Documentation

For detailed documentation, please visit our [Wiki](https://github.com/flowline-io/flowbot/wiki)

## Contributing

Issues and Pull Requests are welcome to help improve the project.

## License

This project is licensed under the [GPL-3.0](LICENSE) License.
