# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/flowline-io/flowbot)](https://goreportcard.com/report/github.com/flowline-io/flowbot)
[![License](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](LICENSE)

Flowbot is an advanced multi-platform chatbot framework that provides intelligent conversation, workflow automation, and comprehensive LLM agent capabilities with extensive third-party integrations.

## Key Features

- **Multi-Platform Chatbot** - Discord, Slack, Tailchat
- **LLM Agent System** - OpenAI-compatible model support, multiple agent types
- **Workflow Engine** - DAG-based execution with 8+ built-in actions
- **MCP Protocol** - Model Context Protocol handler per bot module
- **Message Hub** - Redis Stream pub/sub messaging
- **Scheduling** - Cron jobs, triggers, automated tasks
- **18 Bot Modules** - Extensible module system
- **Admin PWA** - WebAssembly frontend with Fiber v3 backend (go-app/v10)
- **Monitoring** - Prometheus metrics, health probes
- **Security** - OAuth 2.0, API key auth, RBAC

## Architecture

<img src="./docs/architecture/architecture.png" alt="Architecture" align="center" width="100%" />

The system follows a modular architecture:

- **Entry Points**: Server (`cmd`), Agent (`cmd/agent`), Admin PWA (`cmd/app`), Composer CLI (`cmd/composer`)
- **Bot Modules**: 18 specialized handlers (`internal/bots/`)
- **Platform Layer**: Discord, Slack, Tailchat (`internal/platforms/`)
- **Workflow Engine**: DAG execution with step tracking (`internal/bots/workflow/`)
- **Storage**: MySQL + Redis (`internal/store/`)
- **Providers**: 17 third-party integrations (`pkg/providers/`)

## Quick Start

### Requirements

- Go 1.24+
- MySQL database
- Redis server
- [Task](https://taskfile.dev) runner
- Docker (optional)

### Installation

#### Build from Source

```bash
git clone https://github.com/flowline-io/flowbot.git
cd flowbot

# Configure
cp docs/config/config.yaml flowbot.yaml
# Edit flowbot.yaml with your settings

# Build and run
task build
./bin/flowbot
```

#### Docker

```bash
docker build -f deployments/Dockerfile -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot
```

### Initial Setup

1. **Database**: Configure MySQL DSN in `flowbot.yaml`
2. **Redis**: Set Redis connection details
3. **Migrations**: Run `task migrate`
4. **Platform**: Add bot tokens for Discord/Slack/Tailchat
5. **LLM Models**: Configure OpenAI-compatible API endpoint
6. **Start**: Launch server and access Swagger UI at `http://localhost:6060/swagger/`

## Bot Modules

| Module | Description | Features |
|--------|-------------|----------|
| **Agent** | LLM-powered AI | Multiple models, context management |
| **Workflow** | Workflow automation | DAG execution, 8+ actions |
| **Finance** | Financial tracking | Bill tracking, categorization |
| **Kanban** | Project management | Task boards |
| **Notify** | Notifications | Slack, Pushover, ntfy, Message Pusher |
| **Reader** | RSS/Feed reader | Content aggregation |
| **GitHub** | GitHub integration | Issues, PRs |
| **Gitea** | Gitea integration | Repository management |
| **Cloudflare** | Cloudflare | DNS, analytics |
| **Torrent** | Downloads | Transmission integration |
| **Bookmark** | Link management | URL organization, tagging |
| **Search** | Full-text search | MeiliSearch |
| **Clipboard** | Clipboard sync | Cross-platform sync |
| **Anki** | Flashcards | Spaced repetition |
| **Server** | Server management | System operations |
| **Dev** | Developer tools | Debugging, testing |
| **User** | User management | Profiles, settings |
| **Webhook** | Webhooks | Inbound/outbound hooks |

## Development

### Task Runner

```bash
# List all tasks
task -a

# Common checks (tidy → swagger → format → lint → scc)
task default

# Build all binaries
task build:all

# Run with live reload
task air
```

### Build Commands

```bash
task build           # Main server
task build:agent     # Agent
task build:app       # Admin PWA (Wasm + server)
task build:composer  # Composer CLI
```

### Code Generation

```bash
task generator:bot NAME=mybot RULE=command,form  # Generate bot
task generator:vendor NAME=myvendor              # Generate vendor
task dao                                         # Generate DAO from DB
task swagger                                     # Generate Swagger docs
task doc                                         # Generate schema docs
```

### Testing & Quality

```bash
task test           # Run unit tests
task test:all       # Run all tests
task test:coverage  # Coverage report
task lint           # Lint (revive + actionlint)
task check          # All security & quality checks
```

### API

- **Base URL**: `http://localhost:6060/service`
- **Auth**: `X-AccessToken` header
- **Swagger**: `http://localhost:6060/swagger/`
- **Health**: `/livez`, `/readyz`, `/startupz`
- **Metrics**: `/metrics` (Prometheus)

### CLI Tools

```bash
# Composer — code generation & migrations
task generator:bot NAME=mybot RULE=command
task migrate
task migration NAME=add_feature
task workflow:import TOKEN=xxx PATH=./workflow.yaml
```

### Workflow Actions

Built-in actions: **Message**, **Fetch**, **Feed**, **LLM**, **Docker**, **Grep**, **Unique**, **Torrent**

### Third-party Integrations

| Category | Services |
|----------|----------|
| Communication | Discord, Slack, Tailchat |
| Development | GitHub, Gitea, Drone CI |
| Productivity | Kanboard, n8n |
| Finance | Firefly III |
| Infrastructure | AdGuard, Cloudflare, Uptime Kuma |
| Media | Transmission, Miniflux, ArchiveBox, Hoarder |
| Storage | Dropbox |
| Other | Slash, Email |

## Deployment

```bash
# Docker — main server
docker build -f deployments/Dockerfile -t flowbot .

# Docker — admin PWA (multi-stage)
docker build -f deployments/Dockerfile.app -t flowbot-app .

# Systemd agent service
sudo cp docs/deployment/flowbot-agent.service /etc/systemd/system/
sudo systemctl enable flowbot-agent
```

## Configuration

Key sections in `flowbot.yaml`:

```yaml
listen: ":6060"
api_path: "/"

store_config:
  use_adapter: mysql
  adapters:
    mysql:
      dsn: "root:password@tcp(localhost)/flowbot?parseTime=True"

redis:
  addr: "localhost:6379"

models:
  - provider: openai
    base_url: "https://api.openai.com/v1"
    api_key: "your-key"
    model_names: ["gpt-4"]

platform:
  slack:
    enabled: true
    bot_token: "xoxb-..."
  discord:
    enabled: true
    bot_token: "..."
```

## Documentation

- [Project Documentation](docs/README.md)
- [Architecture Guide](docs/architecture/README.md)
- [API Reference](docs/api/README.md)
- [Configuration](docs/config/README.md)
- [Database Schema](docs/database/README.md)
- [Deployment Guide](docs/deployment/README.md)
- [Notification Setup](docs/notify.md)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the [GPL-3.0](LICENSE) License.
