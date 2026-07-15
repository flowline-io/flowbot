# Store Layer

Database storage using Ent with PostgreSQL. Adapter interfaces and higher-level stores live in `store.go`; PostgreSQL method implementations live in `postgres/adapter.go`.

## Structure

```
store/
├── store.go                 # Adapter interfaces, connection management, EventStore/PipelineStore/AuditStore, Migrate()
├── store_test.go            # Store unit tests
├── store_stats_test.go      # Store statistics tests
├── store_token_usage_test.go # Token usage store tests
├── ent/                     # Ent schema definitions and generated code
│   ├── schema/              # Ent schema definitions (tables) + domain types (types.go)
│   └── gen/                 # Ent generated code
├── postgres/                # PostgreSQL adapter
│   ├── adapter.go           # Ent client Adapter method implementations
│   ├── pool.go              # Connection pool with Prometheus metrics
│   └── *_test.go            # Adapter/pool tests + testutil
└── sqlitetest/              # In-memory SQLite test helper
    └── sqlitetest.go
```

## Rules

- Do not create separate `xxx_store.go` files; keep interfaces/facades in `store.go` and SQL/ORM implementations in `postgres/adapter.go`.
- Never write database queries in modules/handlers — call through `store.Database` / store package APIs.
- Test files (`*_test.go`) are co-located with the code under test.

## Patterns

- **Migrations**: Ent auto-migration via `client.Schema.Create()` on startup (`store.Migrate()`). No manual SQL migrations.
- Always use transactions for multi-step operations.
- All ORM operations through ent `gen.Client` (see `internal/store/postgres/adapter.go`).

## Commands

```bash
go tool task webdoc   # Generate schema / web docs via composer
go tool task ent      # Generate ent code from schemas
```
