# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/flowline-io/flowbot)](https://goreportcard.com/report/github.com/flowline-io/flowbot)

**Homelab Data Hub & Capability Orchestration Center**

Flowbot discovers self-hosted apps, abstracts their capabilities, exposes unified interfaces, and orchestrates cross-service automation via declarative Pipelines and Workflows.

## What Flowbot Solves

In a typical homelab, dozens of self-hosted apps run under `/home/<user>/homelab/apps/`. Each has its own API, auth model, pagination convention, and data format. Flowbot answers a single question:

> How do I make all these apps work together?

| Problem                   | Flowbot Solution                                                                                            |
| ------------------------- | ----------------------------------------------------------------------------------------------------------- |
| App discovery & lifecycle | **Homelab Scanner** scans `docker-compose.yaml`, registers apps                                             |
| Capability abstraction    | **Ability Layer** maps apps to unified capabilities (`bookmark`, `archive`, `reader`, ...)                  |
| Unified interfaces        | REST, CLI, Chat, Form, Webhook, Cron, Workflow                                                              |
| Cross-service data flow   | **Declarative Pipeline** — event-driven, idempotent, auditable                                              |
| Composable automation     | **Workflow Capability Step** — DAG of capability invocations                                                |
| Auth boundary             | **AuthContext** spans REST / CLI / Chat / Webhook / Cron / Pipeline / Workflow                              |
| Audit trail               | Durable events, execution history, audit logs — traceable, recoverable, replayable                          |
| Provider differences      | Standard errors (`ErrNotFound`, `ErrForbidden`, `ErrProvider`) + unified pagination (limit + opaque cursor) |

**Flowbot is not a chatbot.** It uses chat as one of many interaction surfaces. At its core, it is a data hub and orchestration engine for your homelab.

## Architecture

```
/home/<user>/homelab/apps
        |                          Module (16 interaction surfaces)
        | scan apps/*/docker-compose.yaml        |
        v                                        v
+-------------------+                  +---------------------+
| Homelab Registry  |  bind app →      | Capability Registry |
| archivebox,atuin, |  capability      | bookmark, archive,  |
| beszel,karakeep...| ---------------> | reader, kanban,     |
+-------------------+                  | infra, shellhistory |
        |                              +---------+-----------+
        | register apps                          |
        v                                ability.Invoke()
+-------------------+                            |
|       Hub         |                            v
| /hub/apps         |                  +--------------------+
| /hub/capabilities |                  |  Ability Layer     |
| /hub/health       |                  |  bookmark.Service  |
+-------------------+                  |  archive.Service   |
                                       |  reader.Service    |
                                       |  kanban.Service    |
                                       |  infra.Service     |
                                       +---------+----------+
                                                 | adapter
                                                 v
                                       +-----------------------+
                                       |  Provider Layer       |
                                       |  karakeep, archivebox,|
                                       |  miniflux, kanboard,  |
                                       |  fireflyiii, beszel,  |
                                       |  atuin, ...           |
                                       +-----------------------+
```

See [architecture diagrams](docs/architecture/README.md) for full PlantUML component, layer, dataflow, and deployment diagrams.

## Capabilities

| Capability        | Apps Mapped                              | Interfaces                     |
| ----------------- | ---------------------------------------- | ------------------------------ |
| **bookmark**      | karakeep, linkwarden                     | REST, CLI, Chat, Workflow      |
| **archive**       | archivebox                               | REST, CLI, Chat, Workflow      |
| **reader**        | miniflux                                 | REST, CLI, Chat, Webhook, Cron |
| **kanban**        | kanboard                                 | REST, CLI, Chat, Webhook       |
| **finance**       | fireflyiii                               | REST, CLI, Chat, Webhook       |
| **infra**         | beszel, uptime-kuma, adguard             | REST, CLI                      |
| **shell_history** | atuin                                    | REST, CLI                      |

All capabilities share the same invocation pattern:

```go
result, err := ability.Invoke(ctx, "bookmark", ability.OpList, ability.Params{Limit: 20})
```

Standard errors, unified pagination, provider-agnostic.

## Pipeline & Workflow

### Declarative Pipeline

Cross-service data flows defined in YAML, triggered by durable events:

```yaml
# When a new bookmark is saved, archive it and notify
trigger:
  event: "bookmark.created"
steps:
  - action: archive.submit
    input: $.event.url
  - action: notify.send
    input:
      channel: slack
      message: "Archived: $.event.url"
```

Every pipeline run is persisted, idempotent, and audited.

### Workflow Capability Step

Composable automation DAG where each step invokes a capability:

```
[cron trigger] → [reader.fetch] → [llm.summarize] → [notify.send]
```

Built-in step types: Capability, Message, Fetch, Feed, LLM, Docker, Grep, Unique, Torrent.

## Quick Start

### Requirements

- Go 1.26+
- MySQL + Redis
- [Task](https://taskfile.dev) runner
- Docker

### Install

```bash
git clone https://github.com/flowline-io/flowbot.git
cd flowbot
cp docs/config/config.yaml flowbot.yaml
# Edit flowbot.yaml
task build
./bin/flowbot
```

### Docker

```bash
docker build -f deployments/Dockerfile -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot
```

## Module Surface

16 modules serve as interaction entry points. Each can expose commands, forms, webhooks, cron jobs, web services, or workflow triggers.

| Module         | Surface                                                        |
| -------------- | -------------------------------------------------------------- |
| **workflow**   | DAG execution, job scheduling                                  |
| **bookmark**   | URL management via capability                                  |
| **archive**    | Web archiving via capability                                   |
| **reader**     | RSS/feed aggregation via capability                            |
| **kanban**     | Task boards via capability                                     |
| **finance**    | Bill tracking via capability                                   |
| **hub**        | App lifecycle management                                       |
| **notify**     | Multi-channel dispatch (Slack, Pushover, ntfy, Message Pusher) |
| **dev**        | Debugging, testing, forms                                      |
| **github**     | Issues, PRs                                                    |
| **gitea**      | Repository management                                          |
| **torrent**    | Transmission integration                                       |
| **search**     | MeiliSearch                                                    |
| **server**     | System operations                                              |
| **user**       | Profiles, settings                                             |
| **webhook**    | Inbound/outbound hooks                                         |

## Development

```bash
task default              # tidy → swagger → format → lint → test
task build                # Main server
task test                 # Unit tests
task lint                 # revive + actionlint
task air                  # Live reload
```

### Code Generation

```bash
task dao       # Generate DAO from database
task swagger   # Generate Swagger/OpenAPI docs
task doc       # Generate database schema docs
```

### API

| Endpoint                       | Description      |
| ------------------------------ | ---------------- |
| `/service/{capability}/*`      | Capability plane |
| `/hub/*`                       | Management plane |
| `/swagger/`                    | OpenAPI docs     |
| `/livez` `/readyz` `/startupz` | Health probes    |
| `/metrics`                     | Prometheus       |

Auth: `X-AccessToken` header or OAuth 2.0.

## Configuration

```yaml
listen: ":6060"
store_config:
  use_adapter: mysql
  adapters:
    mysql:
      dsn: "root:password@tcp(localhost)/flowbot?parseTime=True"
redis:
  addr: "localhost:6379"
platform:
  slack:
    enabled: true
    bot_token: "xoxb-..."
  discord:
    enabled: true
    bot_token: "..."
```

## Documentation

- [Architecture](docs/architecture/README.md)
- [API Reference](docs/api/README.md)
- [Configuration](docs/config/README.md)
- [Database Schema](docs/database/README.md)
- [Deployment](docs/deployment/README.md)
- [Notifications](docs/notify.md)

## License

[GPL-3.0](LICENSE)
