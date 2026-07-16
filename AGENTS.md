# Flowbot

Homelab Data Hub & Capability Orchestration Center.

## Coding guidelines 

* Prioritize code correctness and clarity. Speed and efficiency are secondary priorities unless otherwise specified.
* Prefer implementing functionality in existing files unless it is a new logical component. Avoid creating many small files.
* Do not write organizational or comments that summarize the code. Comments should only be written in order to explain "why" the code is written in some way in the case there is a reason that is tricky / non-obvious.
* Go 1.26.3+, PostgreSQL, Redis required
* Do not use emojis
* Run lint and test after modifying code
* Text in English: comments, docs, commit messages
* Code must have TDD + BDD tests
* In functions, variables, structs, interfaces, etc., must be commented using godoc.
* NEVER git commit unless asked.

## Key Patterns

- **Reference implementations**: When creating or modifying provider, capability, or module code, reference the corresponding `example/` package for file structure and code style:
  - Provider: `pkg/providers/example/` — demonstrates `GetClient()`/`NewXxx()`, OAuth, CRUD, config reading, webhook payload types
  - Capability: `pkg/capability/example/` — demonstrates `Service` interface, `Register()`, `WebhookConverter`, `PollingResource`, conformance, and adapter pattern (`adapter.go`)
  - Module: `internal/modules/example/` — demonstrates `moduleHandler`, `module.Base`, `Register()`, `Init()`, `Rules()`, `Webservice()`, rule definitions
- **Format**: run command `go tool task format`
- **JS Style**: Use single quotes (`'`) for strings
- **Lint**: `revive` (strict, see `revive.toml`)
- **Imports**: stdlib → third-party → internal
- **Naming**: packages lowercase, types CamelCase
- **Errors**: Wrap with `%w`, use `types.ErrNotFound / ErrForbidden / ErrProvider`
- **Pagination**: limit + opaque cursor; provider internals hidden in adapter
- **Routing**: `/service/{module}/*` for module business routes (module name often matches capability/provider ID), `/hub/*` for management
- **AuthContext**: `pkg/auth` subjects are `user` / `token` / `cron` / `pipeline` / `workflow` / `agent` (REST / CLI / Chat are call paths, not subject types)
- **Events**: DataEvent → PostgreSQL `data_events` (+ event outbox) → Redis Stream → pipeline handler → `pipeline_runs`
- **TDD (Test-driven development)**: Red-Green-Refactor cycle. Write test before implementation. `*_test.go` co-located with source. All test functions must use `for _, tt := range tests { t.Run(tt.name, ...) }` pattern. Each table entry must have a descriptive `name` field. Happy path first, error cases required. Single-case tests still wrap in `t.Run`. Each table must contain at least 3 cases. See (docs/testing/tdd-specs.md)
- **BDD (Behavior-Driven Development)**: Ginkgo v2 + Gomega. `Describe`/`Context`/`It` with `SynchronizedBeforeSuite` + `GinkgoParallelProcess()` for per-process database isolation. New modules must include BDD specs. See (docs/testing/bdd-specs.md)
- Use http.NoBody instead of nil in http.NewRequest calls

## Anti-Patterns

- Never use `panic` outside initialization
- Never ignore errors (assign to `_` or handle)
- Never edit generated code
- Never block in event handlers
- Never import `pkg/providers/*` from `internal/modules/*` — use `capability.Invoke`
- Never call provider clients directly in modules
- Never call hub/pipeline/emit DataEvent from inside a provider
- Never return provider-private types from an adapter
- Never write cross-service logic in cron/event handlers — use Pipeline
- Never put hub management APIs under `/service/hub/*` — use `/hub/*` (hub module business routes may still live under `/service/hub`)
- Never hardcode provider names in pipeline/workflow definitions
- Never return 500/400 for all errors — use appropriate status codes
- Never leak provider raw errors or pagination internals to HTTP layer
- Never use Redis Stream as sole event store — persist to PostgreSQL data_events
- Never skip delivery/audit/idempotency records
- Never write database query code outside the `internal/store` package (`store.go` interfaces/facades + `postgres/adapter.go` implementations)
- Never remove `t.Parallel()` to hide test race conditions — fix the root cause instead (e.g. shared-state serialization)
- Never use `encoding/json` Marshal / Unmarshal — use `github.com/bytedance/sonic`. `json.RawMessage` type from stdlib is allowed.

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

## Cursor Cloud specific instructions

Single Go product (server on port `:6060`) plus CLI helpers under `cmd/`. Requires PostgreSQL + Redis. The update script only runs `go mod download`; everything below must be done per session because it is not part of the update script.

### Start services each session (systemd is unavailable)
```bash
sudo pg_ctlcluster 16 main start          # PostgreSQL 16 (data + role/db persist in snapshot)
sudo redis-server --daemonize yes --save "" --requirepass flowbot   # password MUST match flowbot.yaml
```
DB role/database are `flowbot`/`flowbot` (password `flowbot`, superuser). Recreate only if missing:
`sudo -u postgres psql -c "CREATE ROLE flowbot LOGIN PASSWORD 'flowbot' SUPERUSER;" -c "CREATE DATABASE flowbot OWNER flowbot;"`.
Ent auto-migrates on server startup, so no manual migration step is needed.

### Config (`flowbot.yaml`, gitignored, already present at repo root)
Non-obvious validation gotchas (see `pkg/config/config.go` tags / `validate.go`) when deriving config from `docs/reference/config.yaml`:
- `redis.password` must be NON-empty (validator `required,min=1`), so Redis is run with `--requirepass flowbot`.
- Platform `required_if=Enabled true` is **not** uniform: Discord requires app/client/bot credentials; Tailchat requires `api_url`. Slack and Telegram do **not** fail validation with empty tokens — still set unused platforms to `enabled: false` in Cloud.
- Prefer `metrics.enabled: false` when VictoriaMetrics is not running; leaving it on is harmless except push errors.
- `GET /metrics` requires `metrics.bearer_token` or an access token with `admin:metrics` / `admin:*` scope.
- `http.cors.allow_origins` defaults empty (no CORS reflection); `["*"]` never enables credentials. HSTS is sent when `http.tls_behind_proxy` or `modules.web.auth.cookie_secure` is true.
- Local DSN: `store_config.adapters.postgres.dsn` → `postgres://flowbot:flowbot@localhost/flowbot?sslmode=disable`.

### Run / build / lint / test
- Run dev server: `go tool task run` (uses `go run -tags swagger ./cmd`). Health: `/livez`, `/readyz`. Web UI: `/service/web/login` (creds from `modules.web.auth`; reference config uses `admin` / `flowbot-dev-pass`, or set `password_hash`).
- Lint (`go tool task lint`) includes a JS step (`oxlint ./public`); `oxlint` is installed globally via npm. If missing, run `npm install -g oxlint` (npm prefix must point inside the nvm node dir, e.g. `npm config set prefix "$HOME/.nvm/versions/node/v22.22.2"`, and that bin dir must be on PATH).
- Unit tests (`go tool task test`) pass without Docker and use the running Redis.
- `go tool task test:specs` (BDD) needs Docker/testcontainers, which is NOT installed here; install Docker first if you must run them.
