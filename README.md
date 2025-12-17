# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/flowline-io/flowbot)](https://goreportcard.com/report/github.com/flowline-io/flowbot)
[![License](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](LICENSE)

Flowbot is an advanced multi-platform chatbot framework that provides intelligent conversation, workflow automation, and comprehensive LLM agent capabilities with extensive third-party integrations.

## ✨ Key Features

- 🤖 **Multi-Platform Chatbot** - Support for Discord, Slack, Tailchat, and Telegram
- 🧠 **LLM Agent System** - Advanced AI agents with multiple model support (OpenAI, etc.)
- ⚡ **Workflow Engine** - Visual workflow designer with DAG-based execution
- 📨 **Message Hub** - Publish/Subscribe messaging with Redis Stream
- ⏰ **Scheduling System** - Cron jobs, triggers, and automated tasks
- 🔌 **Extensive Integrations** - 20+ built-in bot modules for various services
- � **Monitoring & Metrics** - Prometheus metrics and comprehensive logging
- 🎯 **Multi-tenancy** - Platform-aware user and channel management
- 🔒 **Security** - OAuth integration and role-based access control
- 📱 **Modern Web UI** - Progressive Web App with Fiber v3 backend

## 🏗️ Architecture

<img src="./docs/architecture/architecture.png" alt="Architecture" align="center" width="100%" />

The system follows a modular architecture with:

- **Command Layer**: CLI tools for agent and composer operations
- **Bot Modules**: Specialized handlers for different services
- **Workflow Engine**: Visual workflow design and execution
- **Platform Layer**: Multi-platform messaging support
- **Storage Layer**: MySQL/SQLite with Redis caching

## 🚀 Quick Start

### Requirements

- Go 1.25+
- MySQL/SQLite database
- Redis server
- Docker (optional, for containerized execution)

### Installation

#### Option 1: Build from Source

```bash
# Clone repository
git clone https://github.com/flowline-io/flowbot.git
cd flowbot

# Configure
cp docs/config/config.yaml flowbot.yaml
# Edit flowbot.yaml with your settings

# Build and run
go build -o flowbot ./cmd
./flowbot
```

#### Option 2: Docker

```bash
# Pull and run with docker-compose
docker-compose up -d

# Or build custom image
docker build -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/app/flowbot.yaml flowbot
```

### Initial Setup

1. **Database Setup**: Configure MySQL or SQLite in `flowbot.yaml`
2. **Redis Configuration**: Set Redis connection details
3. **Platform Integration**: Add bot tokens for Discord/Slack/etc.
4. **LLM Models**: Configure OpenAI API key endpoint
5. **Start Service**: Launch flowbot and access Web UI at `http://localhost:6060`

## 🤖 Bot Modules

Flowbot includes 20+ specialized bot modules:

| Module           | Description                   | Features                                   |
| ---------------- | ----------------------------- | ------------------------------------------ |
| **Agent**        | LLM-powered conversational AI | Multiple model support, context management |
| **Workflow**     | Visual workflow automation    | DAG execution, 15+ built-in actions        |
| **Finance**      | Financial data management     | Bill tracking, expense categorization      |
| **Kanban**       | Project management            | Task boards, team collaboration            |
| **Notify**       | Notification system           | Multi-channel alerts, scheduling           |
| **Reader**       | RSS/Feed processing           | Content aggregation, summaries             |
| **GitHub/Gitea** | Git repository integration    | Issue tracking, PR management              |
| **Obsidian**     | Note-taking integration       | Knowledge management, linking              |
| **Torrent**      | Download management           | Transmission integration                   |
| **Bookmark**     | Link management               | URL organization, tagging                  |
| **Search**       | Full-text search              | MeiliSearch integration                    |
| **Clipboard**    | Cross-platform clipboard      | Sync and history                           |

## 🔧 API & Development

### Development Tasks

This repository uses `mise` for common development tasks.

```bash
# list tasks
mise tasks

# run the common checks (tidy/swagger/format/lint/scc)
mise run default

# build all binaries
mise run build:all
```

### RESTful API

- **Base URL**: `http://localhost:6060/service`
- **Authentication**: API key via `X-AccessToken` header
- **Documentation**: Swagger UI available at `/swagger/`
- **Endpoints**: 50+ endpoints for complete system management

### CLI Tools

```bash
# Agent CLI - Desktop automation
./flowbot-agent --config agent.yaml

# Composer CLI - Code generation and migration
./flowbot composer generate bot --name mybot
./flowbot composer migrate import
./flowbot composer workflow import --path workflow.yaml
```

### Workflow Actions

Built-in workflow actions include:

- **Message**: Send notifications to channels
- **Fetch**: HTTP requests and web scraping
- **Feed**: RSS/Atom feed processing
- **LLM**: AI text processing and generation
- **Docker**: Container execution
- **Grep**: Text pattern matching
- **Unique**: Data deduplication
- **Torrent**: File downloads

### Third-party Integrations

Supported services:

- **Communication**: Discord, Slack, Tailchat, Telegram
- **Development**: GitHub, Gitea, Drone CI
- **Productivity**: Notion, Obsidian, Kanboard
- **Finance**: Firefly III, Wallos
- **Infrastructure**: AdGuard, Safeline WAF, Uptime Kuma
- **Media**: Transmission, MiniFlux, ArchiveBox
- **Storage**: MinIO, Dropbox, Hoarder

## 📊 Monitoring & Operations

- **Metrics**: Prometheus integration with custom metrics
- **Logging**: Structured logging with Zerolog
- **Health Checks**: Built-in health and readiness probes
- **Performance**: Automatic GOMAXPROCS tuning
- **Scaling**: Horizontal scaling with Redis clustering

## 🛠️ Configuration

Key configuration sections in `flowbot.yaml`:

```yaml
# Server settings
listen: ":6060"
api_path: "/"

# Database (MySQL/SQLite)
store_config:
  use_adapter: mysql
  adapters:
    mysql:
      dsn: "root:password@tcp(localhost)/flowbot"

# LLM Models
models:
  - provider: openai
    base_url: "https://api.openai.com/v1"
    api_key: "your-key"
    model_names: ["gpt-4", "gpt-3.5-turbo"]

# Platform integrations
platform:
  discord:
    enabled: true
    bot_token: "your-token"
  slack:
    enabled: true
    bot_token: "your-token"
```

## 🚀 Deployment

### Production Deployment

```bash
# Using systemd service
sudo cp docs/flowbot.service /etc/systemd/system/
sudo systemctl enable flowbot
sudo systemctl start flowbot

# Using Docker Compose
docker-compose -f deployments/docker-compose.yml up -d

# Using Kubernetes
kubectl apply -f deployments/k8s/
```

### Environment Variables

```bash
export FLOWBOT_CONFIG_PATH=/path/to/flowbot.yaml
export FLOWBOT_LOG_LEVEL=info
export FLOWBOT_DB_DSN="mysql://user:pass@host/db"
```

## 📚 Documentation

- **API Documentation**: [Swagger UI](http://localhost:6060/swagger/)
- **Configuration Schema**: [docs/schema.md](docs/schema.md)
- **Architecture Guide**: [docs/README.md](docs/README.md)
- **Examples**: [docs/examples/](docs/examples/)
- **Wiki**: [GitHub Wiki](https://github.com/flowline-io/flowbot/wiki)

## 🤝 Contributing

We welcome contributions! Please see our contributing guidelines:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Tools

```bash
# Code generation
task generate

# Linting
revive -formatter friendly ./...

# Security scanning
govulncheck ./...

# Database migrations
migrate -source file://./internal/store/migrate -database "mysql://..." up
```

## 📄 License

This project is licensed under the [GPL-3.0](LICENSE) License - see the LICENSE file for details.

## ⭐ Star History

If you find Flowbot useful, please consider giving it a star on GitHub!
