# Flowbot

Homelab Data Hub & Capability Orchestration Center.

## Dir Reference

| Task             | Location            | Notes                                                 |
| ---------------- | ------------------- | ----------------------------------------------------- |
| Add new module   | `internal/modules/` | See `AGENTS.md` there; reference `modules/example/`   |
| Module framework | `pkg/module/`       | Handler interface                                     |
| Database work    | `internal/store/`   | DAO pattern, all DB queries in store.go, migrations   |
| New provider     | `pkg/providers/`    | See `AGENTS.md` there; reference `providers/example/` |
| Capability layer | `pkg/ability/`      | reference `ability/example/`                          |
| Pipeline engine  | `pkg/pipeline/`     | Event-driven pipelines                                |
| Workflow engine  | `pkg/workflow/`     | Workflow runtime                                      |
| Hub management   | `pkg/hub/`          | App lifecycle                                         |
| Homelab registry | `pkg/homelab/`      | App scanning                                          |
| Authentication   | `pkg/auth/`         | AuthContext helpers                                   |
| Notifications    | `pkg/notify/`       | Multi-channel notify                                  |
| Core types       | `pkg/types/`        | Rulesets, protocol, KV                                |
| API routes       | `internal/server/`  | Fiber v3 handlers                                     |
| Entry points     | `cmd/`              | 3 binaries                                            |
| Frontend/PWA     | `pkg/page/`         | go-app WASM components                                |
| Utilities        | `pkg/utils/`        | Must have unit tests                                  |

## Key Patterns

- **Reference implementations**: When creating or modifying provider, ability, or module code, reference the corresponding `example/` package for file structure and code style:
  - Provider: `pkg/providers/example/` — demonstrates `GetClient()`/`NewXxx()`, OAuth, CRUD, config reading, webhook payload types
  - Ability: `pkg/ability/example/` — demonstrates `Service` interface, `Descriptor()`/`RegisterService()`, `WebhookConverter`, `PollingResource`, conformance, and adapter pattern (`example/adapter.go`)
  - Module: `internal/modules/example/` — demonstrates `moduleHandler`, `module.Base`, `Register()`, `Init()`, `Rules()`, `Webservice()`, rule definitions
- **Format**: `go fmt` + `npx prettier`
- **Lint**: `revive` (strict, see `revive.toml`)
- **Imports**: stdlib → third-party → internal
- **Naming**: packages lowercase, types CamelCase
- **Errors**: Wrap with `%w`, use `types.ErrNotFound / ErrForbidden / ErrProvider`
- **Pagination**: limit + opaque cursor; provider internals hidden in adapter
- **Routing**: `/service/{capability}/*` for business, `/hub/*` for management
- **AuthContext**: REST / CLI / Chat / Webhook / Cron / Pipeline / Workflow
- **Events**: DataEvent → PostgreSQL data_events → Redis Stream → pipeline_runs
- **TDD (Test-driven development)**: Red-Green-Refactor cycle. Write test before implementation. `*_test.go` co-located with source. All test functions must use `for _, tt := range tests { t.Run(tt.name, ...) }` pattern. Each table entry must have a descriptive `name` field. Happy path first, error cases required. Single-case tests still wrap in `t.Run`. Each table must contain at least 3 cases. See (docs/testing/tdd-specs.md)
- **BDD (Behavior-Driven Development)**: Ginkgo v2 + Gomega. `Describe`/`Context`/`It` with `SynchronizedBeforeSuite` + `GinkgoParallelProcess()` for per-process database isolation. New modules must include BDD specs. See (docs/testing/bdd-specs.md)

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
- Never use Redis Stream as sole event store — persist to PostgreSQL data_events
- Never skip delivery/audit/idempotency records
- Never write database query code outside `internal/store/store.go`
- Never use `encoding/json` Marshal / Unmarshal — use `github.com/bytedance/sonic`

## Build & Test, Generate command

```bash
go tool task build            # Main server
go tool task lint             # Code lint
go tool task test             # Unit tests
go tool task test:specs       # BDD acceptance tests (requires Docker)
go tool task test:specs:ci    # BDD with retry + JUnit
go tool task ent              # Generate ent code from database
```

## Configuration

- Runtime: `flowbot.yaml` (copy from `docs/reference/config.yaml`)
- Build: `taskfile.yaml`
- Lint: `revive.toml`
- CI: `.github/workflows/build.yml`

## Notes

- Go 1.26+, PostgreSQL, Redis required
- Do not use emojis
- Run lint and test after modifying code
- Text in English: comments, docs, commit messages
- Code must have TDD + BDD tests
- In functions, variables, structs, interfaces, etc., must be commented using godoc. These comments should explain "what" and "why," without repeating "how.", and should be kept synchronized with the code.
- NEVER git commit unless asked.
