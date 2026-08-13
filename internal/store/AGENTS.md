# Store Layer

Ent + PostgreSQL. Domain query APIs are concrete `*Store` types in package `store`; `postgres` only owns connection lifecycle. Repo-wide standing orders: root [AGENTS.md](../../AGENTS.md).

## Entry points

- `store.go` — connection registry (`Database`, narrow `Adapter`), `Init` / `Migrate`
- `dto.go` — shared list/filter option types used across domain stores
- Domain stores (same package): `user.go`, `platform.go`, `message.go`, `chat.go`, `agent.go`, `file.go`, `module_data.go`, `runtime_agent.go`, `notify_config.go`, `audit.go`, `event.go`, `pipeline.go`, `function.go`, `polling.go`, `workflow.go`, `workflow_run.go`, `hub.go`, `resource_chain.go`, `clip.go`, `gateway.go`, `page_data.go`, `notify.go` (records), `llm_usage.go`, plus `life.go` / `web_account.go` / …
- `postgres/adapter.go`, `postgres/pool.go` — open/close/ping/pool/`GetClient` only
- Schemas: `ent/schema/`; generated: `ent/gen/`
- Tests: co-located `*_test.go`; in-memory helper `sqlitetest/`
- Media handler registry lives in `pkg/media`; this package provides `FileStore` only (injected into media by `internal/server`)

## Boundaries

- Put ORM/query code on domain `*Store` structs that hold `*gen.Client` (pattern: `LifeStore`, `ChatStore`). Do **not** add business CRUD to `Adapter` or `postgres/adapter.go`.
- `PlatformUser` CRUD currently lives on `UserStore` for historical naming; prefer new platform-user APIs on `UserStore` unless doing a deliberate split.
- Call sites (including `internal/modules/web`) use `XxxStoreFromDB()` or `NewXxxStore(client)` — not `store.Database.<BusinessMethod>`.
- `store.Database` / `Adapter` is connection-only: `Open` / `Close` / `IsOpen` / `Ping` / `Stats` / `GetName` / `GetDB` / `GetClient`.
- Prefer `GetClient()` and `XxxStoreFromDB()` over `GetDB().(*Client)`.
- Single-store atomic ops may still use internal `BeginTx` / `WithTx` on the domain store.
- Cross-domain transactions: caller opens `client.Tx`, then `NewXxxStore(tx.Client())` for each domain.
- Do not own media handler registry — that lives in `pkg/media`; server injects `FileStore` into media.
- Migrations: Ent `Schema.Create()` on startup (`store.Migrate()`); no manual SQL migrations.

## Testing / commands

```bash
go tool task ent      # Generate ent from schemas
go tool task webdoc   # Schema / web docs via composer
```
