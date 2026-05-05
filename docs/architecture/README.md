# System Architecture

PlantUML diagrams for Flowbot system architecture. Render with any PlantUML-compatible tool (VS Code extension, plantuml.com, CLI, etc.).

## Diagrams

| File                | Type               | Description                                                        |
| ------------------- | ------------------ | ------------------------------------------------------------------ |
| `architecture.puml` | Component Diagram  | Overall system architecture and component relationships            |
| `layers.puml`       | Layered Diagram    | Abstraction layers from infrastructure to entry points             |
| `dataflow.puml`     | Sequence Diagrams  | Key data flows: chat message, workflow, events, hub, notifications |
| `deployment.puml`   | Deployment Diagram | Docker containers, CI/CD pipelines, external services              |

## Rendering

```bash
# CLI (install plantuml)
plantuml docs/architecture/*.puml

# VS Code: install "PlantUML" extension and preview in-editor
# Online: https://www.plantuml.com/plantuml/uml/
```

## Architecture Overview

### Layers (top to bottom)

```
Layer 6 — External:        Users, Chat Platforms, Third-Party APIs
Layer 5 — Platform:        Discord/Slack/Tailchat adapters
Layer 4 — HTTP Gateway:    Fiber v3 server, REST API, auth middleware
Layer 3 — Business Logic:  16 bot modules, workflow engine, pipeline engine, LLM
Layer 2 — Capability:      ability.Invoke() abstraction over providers
Layer 1 — Providers:       16 third-party service integrations
Layer 0 — Infrastructure:  MySQL, Redis, Docker executor
```

### Management Plane (side plane)

```
Homelab Scanner → App Registry → Hub Manager → Capability Binding → Ability Layer
                                  ↑
                     Discovery Engine (labels + runtime probes)
```

### Key Design Rules

- Modules never import providers directly — use `ability.Invoke()`
- Providers never emit DataEvents, call Hub, or call Pipeline
- Standard pagination: limit + opaque cursor (provider internals hidden)
- Durable events: DataEvent → MySQL data_events → Redis Stream → Pipeline
- All Hub lifecycle operations are audited
- AuthContext spans REST, CLI, Chat, Webhook, Cron, Pipeline, Workflow

### Data Flows

1. **Chat Message**: User → Platform → Adapter → Server → Bot Module → ability.Invoke() → Provider → API
2. **Workflow**: Trigger → Workflow Engine → Executor (Docker) → Pipeline Engine → Notifications
3. **Durable Events**: DataEvent → MySQL (data_events) → Redis Stream Outbox → Pipeline → Actions
4. **Hub Management**: Homelab Scan → Discovery (labels + probes) → App Registry → Hub → Capability Binding → Ability Registry
5. **Notifications**: Module → Dispatcher → [Slack, Pushover, ntfy, Message Pusher]

### Entry Points

| Binary       | Path            | Description                                   |
| ------------ | --------------- | --------------------------------------------- |
| Server       | `cmd/main.go`   | HTTP server (Fiber v3 + fx DI)                |
| Admin CLI    | `cmd/cli/`      | User/token management, config, pipeline admin |
| Composer CLI | `cmd/composer/` | Code generation, schema docs, DAO generation  |

### Bot Modules (16)

archive, bookmark, dev, finance, gitea, github, hub, kanban, notify, reader, search, server, torrent, user, webhook, workflow

### Providers (16)

adguard, archivebox, drone, dropbox, email, fireflyiii, gitea, github, kanboard, karakeep, miniflux, n8n, slack, slash, transmission, uptimekuma

### Notifications (4 channels)

Slack, Pushover, ntfy, Message Pusher

### Shared Packages (32)

ability, alarm, auth, cache, client, config, crawler, event, executor, flog, homelab (with probe/ sub-package), hub, llm, locker, media, migrate, module, notify, page, parser, pipeline, providers, rdb, route, search, stats, types, utils, validate, workflow

### CI/CD (`.github/workflows/`)

| Workflow        | Description        |
| --------------- | ------------------ |
| `build.yml`     | Lint + Build       |
| `testing.yml`   | Run all tests      |
| `build_cli.yml` | Build CLI tools    |
| `docker.yml`    | Build Docker image |
| `release.yml`   | Release pipeline   |
