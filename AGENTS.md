# Flowbot

Homelab Data Hub & Capability Orchestration Center.

## Quick Reference

| Task             | Location            | Notes                         |
| ---------------- | ------------------- | ----------------------------- |
| Add new module   | `internal/modules/` | See `AGENTS.md` there         |
| Module framework | `pkg/module/`       | Handler interface             |
| Database work    | `internal/store/`   | DAO pattern, migrations       |
| New provider     | `pkg/providers/`    | OAuth + API clients           |
| Capability layer | `pkg/ability/`      | ability.Invoke()              |
| Pipeline engine  | `pkg/pipeline/`     | Event-driven pipelines        |
| Workflow engine  | `pkg/workflow/`     | Workflow runtime              |
| Hub management   | `pkg/hub/`          | App lifecycle                 |
| Homelab registry | `pkg/homelab/`      | App scanning                  |
| Authentication   | `pkg/auth/`         | AuthContext helpers           |
| Notifications    | `pkg/notify/`       | Multi-channel notify          |
| Core types       | `pkg/types/`        | Rulesets, protocol, KV        |
| API routes       | `internal/server/`  | Fiber v3 handlers             |
| Entry points     | `cmd/`              | 3 binaries                    |
| Frontend/PWA     | `pkg/page/`         | go-app WASM components        |
| Utilities        | `pkg/utils/`        | Must have unit tests          |

## Key Patterns

- **Format**: `go fmt` + `npx prettier`
- **Lint**: `revive` (strict, see `revive.toml`)
- **Imports**: stdlib → third-party → internal
- **Naming**: packages lowercase, types CamelCase
- **Errors**: Wrap with `%w`, use `types.ErrNotFound / ErrForbidden / ErrProvider`
- **Pagination**: limit + opaque cursor; provider internals hidden in adapter
- **Routing**: `/service/{capability}/*` for business, `/hub/*` for management
- **AuthContext**: REST / CLI / Chat / Webhook / Cron / Pipeline / Workflow
- **Events**: DataEvent → MySQL data_events → Redis Stream → pipeline_runs
- **Testing**: `*_test.go` next to code, table-driven, testify require/assert, `gotestsum`

```bash
go test ./pkg/utils
go test -run ^TestFoo$ ./pkg/utils
go tool task test              # All tests
go tool task test:integration  # Integration tests (requires Docker)
```

## Anti-Patterns

- Never use `panic` outside initialization
- Never ignore errors (assign to `_` or handle)
- Never edit generated code
- Never block in event handlers
- Never import `pkg/providers/*` from `internal/modules/*` — use `ability.Invoke`
- Never call provider clients directly in modules
- Never call hub/pipeline/emit DataEvent from inside a provider
- Never return provider-private types from an adapter
- Never write cross-service logic in cron/event handlers — use Pipeline
- Never mount routes under `/service/hub/*` — use `/hub/*`
- Never hardcode provider names in pipeline/workflow definitions
- Never return 500/400 for all errors — use appropriate status codes
- Never leak provider raw errors or pagination internals to HTTP layer
- Never use Redis Stream as sole event store — persist to MySQL data_events
- Never skip delivery/audit/idempotency records

## Build & CI

```bash
go tool task build   # Main server
go tool task lint    # Code lint
```

## Configuration

- Runtime: `flowbot.yaml` (copy from `docs/config/config.yaml`)
- Build: `taskfile.yaml`
- Lint: `revive.toml`
- CI: `.github/workflows/build.yml`

## Notes

- Go 1.26+, MySQL, Redis required
- Do not use emojis
- Run lint after modifying code
- Text in English: comments, docs, commit messages
- Code must have unit tests
