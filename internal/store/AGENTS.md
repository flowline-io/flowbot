# Store Layer

Ent + PostgreSQL. Interfaces/facades in `store.go`; implementations in `postgres/adapter.go`.

## Entry points

- `store.go` — Adapter APIs, connection, `Migrate()`
- `postgres/adapter.go`, `postgres/pool.go`
- Schemas: `ent/schema/`; generated: `ent/gen/`
- Tests: co-located `*_test.go`; in-memory helper `sqlitetest/`

## Boundaries

- Do not add `xxx_store.go` facades — keep interfaces in `store.go`, SQL/ORM in `postgres/adapter.go`
- Never write DB queries in modules/handlers — use `store.Database` / store package APIs
- Migrations: Ent `Schema.Create()` on startup (`store.Migrate()`); no manual SQL migrations
- Multi-step ops use transactions; ORM via `gen.Client`

## Testing / commands

```bash
go tool task ent      # Generate ent from schemas
go tool task webdoc   # Schema / web docs via composer
```
