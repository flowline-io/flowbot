# Store Layer

Ent + PostgreSQL. Interfaces/facades in `store.go`; implementations in `postgres/adapter.go`.

## Entry points

- `store.go` — Adapter APIs, connection, `Migrate()`, most store facades
- `life.go` — `LifeStore` domain facade (same package)
- `postgres/adapter.go`, `postgres/pool.go`
- Schemas: `ent/schema/`; generated: `ent/gen/`
- Tests: co-located `*_test.go`; in-memory helper `sqlitetest/`

## Boundaries

- Do not add separate `*_store.go` facade packages or split the public store API across packages — keep facades in package `store`
- Large domain stores may live in dedicated files in the same package (e.g. `life.go` for `LifeStore`) when `store.go` would become unwieldy; ORM still uses `gen.Client` here (same pattern as other stores in `store.go`)
- Never write DB queries in modules/handlers — use `store.Database` / store package APIs
- Migrations: Ent `Schema.Create()` on startup (`store.Migrate()`); no manual SQL migrations
- Multi-step ops use transactions; ORM via `gen.Client`

## Testing / commands

```bash
go tool task ent      # Generate ent from schemas
go tool task webdoc   # Schema / web docs via composer
```
