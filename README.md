# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)

**Homelab data hub — discover apps, invoke capabilities, automate across them**

Flowbot scans your self-hosted stack, wraps each integrated service behind one `capability.Invoke` API, and runs cross-app automation with Pipelines, Workflows, and Agents — from the Web UI, REST, CLI, or chat.

## What Flowbot Solves

A typical homelab runs dozens of apps under `/home/<user>/homelab/apps/`. Each has its own API, auth, and data shape — wiring them together usually means one-off scripts and fragile glue. Flowbot answers one question:

> How do I make these apps work together — from one place, with a trail I can trust?

| Pain                                         | Flowbot                                                                                          |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Don't know what's running or healthy         | **Homelab Scanner + Hub** — discover from `docker-compose.yaml`, inspect apps / capabilities / health |
| Every service has a different API            | **Capability Layer** — one `capability.Invoke` for `karakeep`, `miniflux`, `gitea`, …           |
| Cross-app flows are ad-hoc scripts           | **Pipelines** (event-driven) and **Workflows** (capability / docker / shell / machine DAGs)      |
| Need Web, CLI, chat, cron, and webhooks      | Same capabilities on every surface — plus Agents when you want a conversational loop             |
| Hard to audit or replay what ran             | Durable events, execution history, and audit logs                                                |

## Web UI

![Flowbot Web UI — Home](docs/home-page.png)

## Architecture

```
/home/<user>/homelab/apps
        |                          Module (interaction surfaces)
        | scan apps/*/docker-compose.yaml        |
        v                                        v
+-------------------+                  +---------------------+
| Homelab Registry  |  bind app →      | Capability Registry |
| archivebox,atuin, |  capability      | karakeep, miniflux, |
| adguard,karakeep… | ---------------> | kanboard, gitea, …  |
+-------------------+                  | notify, agent       |
        |                              +---------+-----------+
        | register apps                          |
        v                                capability.Invoke()
+-------------------+                            |
|       Hub         |                            v
| /hub/apps         |                  +--------------------+
| /hub/capabilities |                  | Capability Layer   |
| /hub/health       |                  | pkg/capability/*   |
+-------------------+                  | karakeep.Service   |
                                       | miniflux.Service   |
                                       | …                  |
                                       +---------+----------+
                                                 | adapter
                                                 v
                                       +-----------------------+
                                       |  Provider Layer       |
                                       |  pkg/providers/*      |
                                       +-----------------------+
```

Layers (top → bottom): Platform adapters → HTTP gateway (Fiber) → modules / pipeline / workflow / agent → `capability.Invoke` → providers → PostgreSQL + Redis.

See [architecture diagrams](docs/architecture/README.md) for PlantUML component, layer, dataflow, and deployment diagrams. Agent engine details live under [docs/agent/](docs/agent/).

## Capabilities

### Supported (full Capability layer)

Provider-backed capabilities use the provider ID as the capability name. Domain event names (e.g. `bookmark.created`) stay stable for orchestration.

| Capability   | Kind              | Notes                                              |
| ------------ | ----------------- | -------------------------------------------------- |
| **karakeep** | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **miniflux** | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **kanboard** | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **trilium**  | Provider          | REST, CLI, Chat, Workflow; polling event source    |
| **memos**    | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **fireflyiii** | Provider        | REST, CLI, Chat, Workflow                          |
| **transmission** | Provider      | REST, CLI, Chat, Workflow                          |
| **gitea**    | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **github**   | Provider          | REST, CLI, Chat, Workflow, Webhook                 |
| **nocodb**   | Provider          | REST, CLI, Chat, Workflow                          |
| **devops**   | Aggregator        | Multi-provider ops (beszel, uptimekuma, …)         |
| **clip**     | Provider          | REST, CLI, Chat, Workflow                          |
| **notify**   | Internal          | Multi-channel dispatch (Slack, Pushover, ntfy, …)  |
| **agent**    | Internal          | Chat / Cloud Agent loop (`pkg/agent/`)             |

### Discovery / client only

These packages live under `pkg/providers/` for Homelab discovery or OAuth/client helpers. They are **not** full Capability integrations (no `capability.Invoke` service surface):

archivebox, adguard, uptimekuma, drone, dropbox, email, n8n, slash, slack (OAuth), grafana, beszel, dozzle, netalertx, wakapi, traefik, …

All supported capabilities share the same invocation pattern:

```go
result, err := capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
```

Standard errors, unified pagination, provider adapters behind `pkg/capability/<provider>/`.

## Pipeline & Workflow

### Declarative Pipeline

Cross-service data flows stored in PostgreSQL (apply via `flowbot pipeline apply` or the Web UI), triggered by durable events:

```yaml
# When a new bookmark is saved, notify
name: bookmark_notify
enabled: true
triggers:
  - type: event
    enabled: true
    event: bookmark.created
steps:
  - name: send_notification
    capability: core
    operation: notify_send
    params:
      template_id: "bookmark.saved"
      channels: ["slack"]
      payload:
        url: "{{event.url}}"
```

Every pipeline run is persisted, idempotent, and audited. Events flow: DataEvent → PostgreSQL `data_events` → Redis Stream → pipeline handler → `pipeline_runs`.

### Workflow Engine

Composable task DAGs in YAML. Each task uses an action prefix:

```
[cron trigger] → [capability:miniflux.list_entries] → [mapper:] → [capability:core.notify_send]
```

| Prefix        | Runtime               | Example                      |
| ------------- | --------------------- | ---------------------------- |
| `capability:` | Capability invoke     | `capability:karakeep.create` |
| `docker:`     | Docker container      | `docker:nginx:latest`        |
| `shell:`      | Shell command         | `shell:echo hello`           |
| `machine:`    | Remote SSH            | `machine:vm1`                |
| `mapper:`     | Inline data transform | `mapper:`                    |

## Quick Start

### Requirements

- Go 1.26.5+
- PostgreSQL + Redis
- [Task](https://taskfile.dev) runner (`go tool task`)
- Docker (for BDD specs / workflow docker steps)

### Install

```bash
git clone https://github.com/flowline-io/flowbot.git
cd flowbot
cp docs/reference/config.yaml flowbot.yaml
# Edit flowbot.yaml — set postgres.dsn and redis.url
go tool task build
./bin/flowbot
```

Or run without building:

```bash
go tool task run
```

Health probes: `/livez`, `/readyz`, `/startupz`. Web UI: `/service/web/login`.

### Docker

Single-node compose (PostgreSQL + Redis + Flowbot):

```bash
cp docs/reference/config.yaml deployments/flowbot.yaml
# Edit deployments/flowbot.yaml (DSN, redis URL, web auth)
cd deployments && docker compose up -d --build
```

See [Self-hosting](docs/self-hosting.md) for reverse-proxy, backups, and security checklist.

Or build the image alone:

```bash
docker build -f deployments/Dockerfile -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot
```

### CLI

Install the `flowbot` CLI from GitHub releases:

```bash
curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.sh | bash
```

Or install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.sh | bash -s -- --version v0.40
```

The CLI is installed to `/usr/local/bin/flowbot`. Run `flowbot --help` to see available commands. From source: `go tool task build:cli` or `go install github.com/flowline-io/flowbot/cmd/cli@latest`.

## Modules & Platforms

Interaction modules are thin entry points (commands, webhooks, webservice, cron). They never import `pkg/providers/*` — they call `capability.Invoke`.

| Module      | Surface                                              |
| ----------- | ---------------------------------------------------- |
| **hub**     | App / capability lifecycle, management APIs (`/hub/*`) |
| **web**     | Web UI, login, service routes (`/service/web/*`)     |
| **example** | Reference module for new modules                     |

Chat platforms: **Discord**, **Slack**, **Tailchat** (`internal/platforms/`).

Binaries: server (`cmd/`), admin CLI (`cmd/cli/`), composer (`cmd/composer/` — admin actions, website docs, SKILL.md generation).

## Development

```bash
go tool task default       # tidy → swagger → format → lint → scc
go tool task build         # Main server → bin/flowbot
go tool task run           # go run -tags swagger ./cmd
go tool task test          # Unit tests
go tool task test:specs    # BDD acceptance tests (Docker required)
go tool task lint          # revive + testify + actionlint + oxlint
go tool task air           # Live reload
```

### Code Generation

```bash
go tool task swagger   # Generate Swagger/OpenAPI docs
go tool task ent       # Generate ent code from database
go tool task templ     # Generate Go code from Templ templates
go tool task skills    # Generate SKILL.md for CLI
```

### API

| Endpoint                       | Description      |
| ------------------------------ | ---------------- |
| `/service/{capability}/*`      | Capability plane |
| `/hub/*`                       | Management plane |
| `/swagger/`                    | OpenAPI docs     |
| `/livez` `/readyz` `/startupz` | Health probes    |
| `/metrics`                     | Prometheus       |

Auth: `X-AccessToken` header or OAuth 2.0. Service routes require minimum scopes (`service:{capability}:read|write`, etc.).

## Configuration

```yaml
listen: ":6060"
postgres:
  dsn: "postgres://flowbot:flowbot@localhost/flowbot?sslmode=disable"
redis:
  url: "redis://:flowbot@127.0.0.1:6379/0"
platform:
  slack:
    enabled: false
  discord:
    enabled: false
  tailchat:
    enabled: false
```

Full template: [`docs/reference/config.yaml`](docs/reference/config.yaml). Field reference: [`docs/reference/config-reference.md`](docs/reference/config-reference.md).

## Documentation

- [Getting Started](docs/getting-started/README.md)
- [User Guide](docs/user-guide/README.md) — pipelines, workflows, notifications, homelab discovery
- [Architecture](docs/architecture/README.md)
- [API Reference](docs/api/README.md)
- [Configuration](docs/reference/config-reference.md)
- [Database Schema](docs/reference/database-reference.md)
- [Deployment](docs/developer-guide/deployment.md)
- [Agent Engine](docs/agent/README.md)
- [Developer Guide](docs/developer-guide/README.md)
- [Testing](docs/testing/README.md)

## License

[GPL-3.0](LICENSE)
