# Agents Guide for Flowbot

Flowbot is Homelab Data Hub & Capability Orchestration Center

**Generated:** 2026-05-01

## Quick Reference

| Task             | Location            | Notes                   |
| ---------------- | ------------------- | ----------------------- |
| Add new module   | `internal/modules/` | See `AGENTS.md` there   |
| Module framework | `pkg/module/`       | Handler interface       |
| Database work    | `internal/store/`   | DAO pattern, migrations |
| New provider     | `pkg/providers/`    | OAuth + API clients     |
| Capability layer | `pkg/ability/`      | ability.Invoke()        |
| Pipeline engine  | `pkg/pipeline/`     | Event-driven pipelines  |
| Workflow engine  | `pkg/workflow/`     | Workflow runtime        |
| Hub management   | `pkg/hub/`          | App lifecycle           |
| Homelab registry | `pkg/homelab/`      | App scanning            |
| Authentication   | `pkg/auth/`         | AuthContext helpers     |
| Notifications    | `pkg/notify/`       | Multi-channel notify    |
| Core types       | `pkg/types/`        | Rulesets, protocol, KV  |
| API routes       | `internal/server/`  | Fiber v3 handlers       |
| Entry points     | `cmd/`              | 3 binaries              |
| Frontend/PWA     | `pkg/page/`         | go-app WASM components  |
| Utilities        | `pkg/utils/`        | Must have unit tests    |

## Structure

```
flowbot/
├── cmd/                  # Entry points
│   ├── main.go          # HTTP server (Fiber)
│   ├── composer/        # CLI: dao gen, schema doc
│   └── cli/             # CLI: admin commands
├── internal/
│   ├── modules/         # 20 bot modules
│   ├── server/          # Fiber v3 HTTP layer
│   ├── store/           # GORM DAO/models
│   └── platforms/       # Discord, Slack, Tailchat
├── pkg/
│   ├── types/           # Core type system
│   ├── providers/       # 17 third-party integrations
│   ├── page/            # PWA frontend (go-app/WASM)
│   ├── utils/           # Common utilities
│   ├── event/           # Redis Stream pub/sub
│   ├── executor/        # Workflow runtime (Docker)
│   ├── llm/             # LLM functinon
│   ├── chatbot/         # Platform chat interface
│   ├── migrate/         # Migration runner
│   ├── ability/         # Capability abstraction layer
│   ├── pipeline/        # Event-driven pipelines
│   ├── workflow/        # Workflow engine
│   ├── hub/             # Hub management
│   ├── homelab/         # Homelab app registry
│   ├── module/          # Module framework
│   ├── auth/            # Authentication helpers
│   ├── notify/          # Multi-channel notifications
│   ├── config/          # Configuration
│   ├── flog/            # Structured logging
│   ├── rdb/             # Redis helpers
│   ├── stats/           # Metrics
│   ├── route/           # HTTP routing utilities
│   ├── parser/          # Command parser
│   ├── locker/          # Distributed locking
│   ├── cache/           # Caching layer
│   ├── media/           # Media handling
│   ├── client/          # API clients
│   ├── alarm/           # Alarm/scheduling
│   ├── crawler/         # Web crawler
│   ├── search/          # Search utilities
│   └── validate/        # Validation
```

## Architecture

```
/home/<user>/homelab/apps
        |
        | scan apps/*/docker-compose.yaml
        v
+--------------------------------------------------+
|                  Homelab App Registry            |
|  archivebox | atuin | beszel | flowbot | ...     |
+-------------------------+------------------------+
                          |
                          | bind app -> capability
                          v
+--------------------------------------------------+
|                  Hub                             |
|  /hub/apps                                       |
|  /hub/capabilities                               |
|  /hub/health                                     |
|  /hub/apps/:name/restart                         |
+-------------------------+------------------------+
                          |
                          | register capabilities
                          v
+--------------------------------------------------+
|              Capability Registry                 |
|  bookmark | archive | reader | kanban | infra    |
+-------------------------+------------------------+
                          |
                          | ability.Invoke()
                          v
+--------------------------------------------------+
|              Ability Layer                       |
|  bookmark.Service                                |
|  archive.Service                                 |
|  reader.Service                                  |
|  kanban.Service                                  |
|  infra.Service                                   |
+-------------------------+------------------------+
                          |
                          | adapter
                          v
+--------------------------------------------------+
|              Provider Layer                      |
|  karakeep | linkwarden | archivebox | miniflux   |
|  kanboard | fireflyiii | beszel | atuin | ...    |
+--------------------------------------------------+
```

### Cross-cutting

**AuthContext**: REST / CLI / Chat / Webhook / Cron / Pipeline / Workflow

**Standard Error**:

```go
errors.Is(err, types.ErrNotFound)
errors.Is(err, types.ErrForbidden)
errors.Is(err, types.ErrProvider)
```

**Standard Pagination**: limit + opaque cursor, provider page/offset/token hidden in adapter

**Durable Event**:

```
DataEvent → MySQL data_events
EventOutbox → Redis Stream
Pipeline Engine → pipeline_runs
```

**Store**: `internal/store` → MySQL, migrations, GORM Gen

## Key Patterns

### Code Style

- **Format**: `go fmt` + `npx prettier`
- **Lint**: `revive` (strict, see `revive.toml`)
- **Imports**: stdlib → third-party → internal
- **Naming**: packages lowercase, types CamelCase
- **Errors**: Wrap with `%w`, use `errors.New` for sentinels

### Routing

```
/service/{capability}/*  # Business capability plane
/hub/*                    # Management plane
```

### Testing

```bash
go test ./pkg/utils
go test -run ^TestFoo$ ./pkg/utils

go tool task test          # All tests
go tool task test:integration # Integration tests (requires Docker)
```

- Tests live next to code: `*_test.go`
- Use `require`/`assert` from testify
- Table-driven tests preferred
- Run with `gotestsum` (installed as Go tool)

## Anti-Patterns

- **Never** use `panic` outside initialization
- **Never** ignore errors (assign to `_` or handle)
- **Never** edit generated code directly
- **Never** block in event handlers
- **Always** check `err != nil` immediately
- **Always** wrap errors when propagating
- **Never** import `pkg/providers/*` directly from `internal/modules/*` — use `ability.Invoke`
- **Never** call provider clients directly in modules (e.g. `karakeep.GetClient()`)
- **Never** access local filesystem paths (e.g. `/home/<user>/homelab/apps/<app>/data`) from modules
- **Never** call hub or pipeline from inside a provider
- **Never** emit `DataEvent` from inside a provider
- **Never** return provider-private types from an adapter
- **Never** write cross-service logic in cron handlers — use Pipeline
- **Never** write cross-service logic in event handlers — use Pipeline
- **Never** mount routes under `/service/hub/*` — use `/hub/*` for management plane
- **Never** allow arbitrary shell execution in Hub lifecycle
- **Never** access paths outside `apps_dir` in Hub
- **Never** enable exec/update/pull by default
- **Never** hardcode provider names in workflow definitions
- **Never** hardcode provider names in pipeline definitions
- **Never** use app name instead of capability name for `/service` routes
- **Never** return 500 for all errors — use appropriate status codes
- **Never** return 400 for all errors — use appropriate status codes
- **Never** leak provider raw errors to HTTP layer — wrap with adapter standard errors
- **Never** expose provider pagination internals to API callers
- **Never** expose cursor/offset/page internal structure in API responses
- **Never** use Redis Stream as sole long-term event store — persist to MySQL `data_events` table
- **Never** discard Pipeline execution history — write to database
- **Never** skip delivery record on webhook triggers — save delivery log
- **Never** store API tokens in plaintext
- **Never** skip audit log on Hub lifecycle operations
- **Never** skip idempotency checks in Pipeline steps

## Build Commands

```bash
go tool task build         # Main server
```

## CI/Quality

```bash
go tool task lint      # code lint
```

## Configuration

- Runtime: `flowbot.yaml` (copy from `docs/config/config.yaml`)
- Build: `taskfile.yaml`
- Lint: `revive.toml`
- CI: `.github/workflows/build.yml`

## Notes

- Go 1.26+ required
- MySQL + Redis required
- Do not use emojis
- You must run lint after modifying the code.
- Code comments, documentation, Git commit messages, and other text should all be written in English.
- The code must have unit tests.
