# Flowbot Documentation

Flowbot is a Homelab Data Hub & Capability Orchestration Center — it discovers self-hosted apps, abstracts their capabilities, exposes unified interfaces, and orchestrates cross-service automation.

## Directory Structure

- [api/](./api/) — API documentation
  - `swagger.json` / `swagger.yaml` — OpenAPI 3.0 spec
  - `docs.go` — Auto-generated swagger embedding
  - `api.http` — HTTP request examples

- [architecture/](./architecture/) — System architecture (PlantUML)
  - `architecture.puml` — Component diagram
  - `layers.puml` — Layered architecture
  - `dataflow.puml` — Data flow sequences
  - `deployment.puml` — Deployment diagram

- [config/](./config/) — Configuration templates
  - `config.yaml` — Main server config
  - `agent.yaml` — Desktop agent config

- [database/](./database/) — Database documentation
  - `schema.md` — Full table schema reference

- [deployment/](./deployment/) — Deployment guides
  - `flowbot.service` — systemd unit

- [examples/workflows/](./examples/workflows/) — Workflow examples
  - `save_and_track.yaml` — Capability pipeline example

- [conformance.md](./conformance.md) — Ability adapter conformance test suite
- [notify.md](./notify.md) — Notification configuration
- [pipeline.md](./pipeline.md) — Pipeline engine: retry, checkpointing, recovery
- [pipeline-template.md](./pipeline-template.md) — Pipeline template engine reference
- [tracing.md](./tracing.md) — OpenTelemetry distributed tracing
- [workflow.md](./workflow.md) — Workflow engine: retry, DAG, persistent state
- [recovery.md](./recovery.md) — Restart recovery for incomplete pipeline/workflow runs

## Development Tools

All tools are managed as Go tools (`go get -tool`) or through [Task](https://taskfile.dev).

### Task Runner

```bash
task -a              # List all tasks
task default         # tidy → swagger → format → lint → test
task build:all       # Build all binaries
```

### Build Commands

```bash
go tool task build           # Main server
go tool task build:composer  # Composer CLI
go tool task build:cli       # Admin CLI
go tool task build:all       # All binaries
go tool task air             # Live reload
```

### Code Generation

```bash
go tool task dao       # Generate DAO from database
go tool task swagger   # Generate Swagger/OpenAPI docs
go tool task doc       # Generate database schema docs
```

### Workflow CLI

```bash
go run ./cmd/cli workflow run ./docs/examples/workflows/save_and_track.yaml
```

### Code Quality

```bash
go tool task lint      # revive + actionlint
go tool task format    # go fmt + prettier
go tool task tidy      # go mod tidy
```

### Security

```bash
go tool task secure    # govulncheck
go tool task leak      # gitleaks
go tool task gosec     # security scan
go tool task check     # all security & quality
```

### Testing

```bash
go tool task test            # All unit tests
go tool task test:short      # Short mode (skip integration)
go tool task test:utils      # pkg/utils only
go tool task test:integration # Integration tests (Docker)
go tool task test:coverage   # Coverage report
```

#### Conformance Tests

New provider adapters must pass the ability conformance suite:

```bash
go test ./pkg/ability/...                           # All ability + conformance tests
go test -run TestConformance ./pkg/ability/bookmark/karakeep/  # Run single adapter
go test ./pkg/ability/conformance/                   # Conformance framework self-tests
```

### Add Go Tool Dependency

```bash
go get -tool import_path@version
```

## Contributing

1. Fork this project
2. Create a feature branch
3. Commit your changes
4. Create a Pull Request

## License

GPL 3.0 — see [LICENSE](../LICENSE).
