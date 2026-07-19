# Flowbot

[![Build](https://github.com/flowline-io/flowbot/actions/workflows/build.yml/badge.svg)](https://github.com/flowline-io/flowbot/actions/workflows/build.yml)

**Homelab Data Hub & Capability Orchestration Center**

Flowbot discovers self-hosted apps, exposes a unified invocation surface per integrated provider, and orchestrates cross-service automation via declarative Pipelines and Workflows.

## What Flowbot Solves

In a typical homelab, dozens of self-hosted apps run under `/home/<user>/homelab/apps/`. Each has its own API, auth model, pagination convention, and data format. Flowbot answers a single question:

> How do I make all these apps work together?

| Problem                   | Flowbot Solution                                                                                            |
| ------------------------- | ----------------------------------------------------------------------------------------------------------- |
| App discovery & lifecycle | **Homelab Scanner** scans `docker-compose.yaml`, registers apps                                             |
| Capability abstraction    | **Capability Layer** exposes each integrated provider (`karakeep`, `miniflux`, …) via `capability.Invoke` |
| Unified interfaces        | REST, CLI, Chat, Webhook, Cron, Workflow, Agent                                                             |
| Cross-service data flow   | **Declarative Pipeline** — event-driven, idempotent, auditable                                              |
| Composable automation     | **Workflow Engine** — DAG of capability / docker / shell / machine steps                                    |
| Auth boundary             | **AuthContext** subjects: `user` / `token` / `cron` / `pipeline` / `workflow` / `agent`                    |
| Audit trail               | Durable events, execution history, audit logs — traceable, recoverable, replayable                          |
| Provider differences      | Standard errors (`ErrNotFound`, `ErrForbidden`, `ErrProvider`) + unified pagination (limit + opaque cursor) |

**Flowbot is not a chatbot.** It uses chat (Discord / Slack / Tailchat) as one of many interaction surfaces. At its core, it is a data hub and orchestration engine for your homelab.

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
| **notify**   | Internal          | Multi-channel dispatch (Slack, Pushover, ntfy, …)  |
| **agent**    | Internal          | Chat / Cloud Agent loop (`pkg/agent/`)             |

Providers without a capability package yet (discovery / client only): archivebox, adguard, uptimekuma, drone, dropbox, email, n8n, slash, slack (OAuth).

All capabilities share the same invocation pattern:

```go
result, err := capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
```

Standard errors, unified pagination, provider adapters behind `pkg/capability/<provider>/`.

See [UPGRADE-capability-1to1.md](docs/migrations/UPGRADE-capability-1to1.md) when migrating from domain CapTypes (`bookmark`, `reader`, …).

## Pipeline & Workflow

### Declarative Pipeline

Cross-service data flows defined in `pipelines.yaml`, triggered by durable events:

```yaml
# When a new bookmark is saved, notify
- name: bookmark_notify
  enabled: true
  trigger:
    event: bookmark.created
  steps:
    - name: send_notification
      capability: notify
      operation: send
      params:
        channel: slack
        message: "Saved: {{event.url}}"
```

Every pipeline run is persisted, idempotent, and audited. Events flow: DataEvent → PostgreSQL `data_events` → Redis Stream → pipeline handler → `pipeline_runs`.

### Workflow Engine

Composable task DAGs in YAML. Each task uses an action prefix:

```
[cron trigger] → [capability:miniflux.list_entries] → [mapper:] → [capability:notify.send]
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
- [Testing](docs/testing/tdd-specs.md)

## License

[GPL-3.0](LICENSE)
