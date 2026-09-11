# System Architecture

PlantUML diagrams for Flowbot system architecture. Render with any PlantUML-compatible tool (VS Code extension, plantuml.com, CLI, etc.).

## Diagrams

| File                | Type               | Description                                                        |
| ------------------- | ------------------ | ------------------------------------------------------------------ |
| `architecture.puml` | Component Diagram  | Overall system architecture and component relationships            |
| `layers.puml`       | Layered Diagram    | Abstraction layers from infrastructure to entry points             |
| `dataflow.puml`     | Sequence Diagrams  | Key data flows: chat message, workflow, events, hub, notifications |
| `deployment.puml`   | Deployment Diagram | Docker containers, CI/CD pipelines, external services              |

Agent engine (`pkg/agent/`) has dedicated docs and diagrams under [docs/agent/](../agent/) (`architecture.md`, `agent.puml`).

Life (solo gamified productivity) has dedicated docs under [docs/life/](../life/) (`architecture.md`).

pkg vs internal dependency boundaries (migration waves): [pkg-boundaries.md](pkg-boundaries.md).

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
Layer 3 — Business Logic:  modules, workflow engine, pipeline engine, LLM, agent engine
Layer 2 — Capability:      capability.Invoke() abstraction over providers
Layer 1 — Providers:       third-party service integrations (pkg/providers)
Layer 0 — Infrastructure:  PostgreSQL, Redis, Docker/kern executor
```

### Management Plane (side plane)

```
Homelab Scanner → App Registry → Hub Manager → Capability Binding → Capability Layer
                                  ↑
                     Discovery Engine (labels + runtime probes)
```

### Standing orders (do not restate)

Root [AGENTS.md](../../AGENTS.md). Pagination: [pkg/capability/AGENTS.md](../../pkg/capability/AGENTS.md). pkg vs internal: [pkg-boundaries.md](pkg-boundaries.md). Auth call paths: [pkg/auth/AGENTS.md](../../pkg/auth/AGENTS.md). Hub lifecycle audit: [internal/server/AGENTS.md](../../internal/server/AGENTS.md).

### Data Flows

1. **Chat Message**: User → Platform → Adapter → Server → Module → capability.Invoke() → Provider → API
2. **Workflow**: Trigger → Automate module → Workflow Engine → Executor (Docker/kern) → capability steps → Notifications
3. **Durable Events**: DataEvent → PostgreSQL (data_events) → Redis Stream Outbox → Pipeline → Actions
4. **Hub Management**: Homelab Scan → Discovery (labels + probes) → App Registry → Hub → Capability Binding
5. **Notifications**: Module / CapCore notify_send → Dispatcher → [Slack, Pushover, ntfy, Message Pusher, inapp]

### Entry Points

| Binary       | Path            | Description                                            |
| ------------ | --------------- | ------------------------------------------------------ |
| Server       | `cmd/main.go`   | HTTP server (Fiber v3 + fx DI)                         |
| Admin CLI    | `cmd/cli/`      | User/token management, config, pipeline/workflow admin |
| Composer CLI | `cmd/composer/` | Admin actions, website docs, and SKILL.md generation   |
| Gateway      | `cmd/gateway/`  | Local CLI gateway worker (pull model)                  |
| Agent CLI    | `cmd/agent/`    | Headless coding agent (`flowbot-agent`)                |

### Modules (5)

automate, example, hub, life, web

`automate` owns REST under `/service/automate/{functions,pipeline,workflow}` (scopes remain `function:*` / `pipeline:*` / `workflow:*`). See [merge note](../../.agents/notes/implemented/simplification/2026-09-05-merge-automate-modules.md).

### Providers (29)

adguard, archivebox, beszel, confluence, dozzle, drone, dropbox, email, example, fireflyiii, gitea, github, grafana, kanboard, karakeep, memos, miniflux, n8n, netalertx, nocodb, scanopy, slack, slash, traefik, transmission, trello, trilium, uptimekuma, wakapi

### Capability packages (`pkg/capability/`)

confluence, core, devops, email, example, fireflyiii, functions, gateway, gitea, github, kanboard, karakeep, life, memos, miniflux, nocodb, transmission, trello, trilium — plus `conformance/`

### Notifications

Outbound: Slack, Pushover, ntfy, Message Pusher (with `rules/` throttling/aggregation and `template/`). In-app: `pkg/notify/inapp`. CapCore `notify_send` is the pipeline/capability entry; see [notification-gateway.md](../user-guide/notification-gateway.md).

### Shared Packages (37)

agent, auth, backoff, bulkhead, cache, capability, client, config, cronutil, event, exec, executor, flog, functions, homelab (with probe/), hub, i18n, life, media, metrics, module, notify, parser, pipeline, plugin, profiling, providers, rdb, route, stats, trace, types, utils, validate, views, webauth, workflow

### CI/CD (`.github/workflows/`)

| Workflow                   | Description                              |
| -------------------------- | ---------------------------------------- |
| `build.yml`                | Lint + Build                             |
| `testing.yml`              | Run all tests                            |
| `build_cli.yml`            | Build CLI tools                          |
| `build_agent.yml`          | Build headless agent binary              |
| `build_gateway.yml`        | Build gateway worker binary              |
| `docker.yml`               | Build Docker image                       |
| `docker-agent-sandbox.yml` | Build agent sandbox image                |
| `agent-eval.yml`           | Agent evaluation (regression)            |
| `release.yml`              | Release pipeline                         |
| `pages.yml`                | Publish website to GitHub Pages          |
