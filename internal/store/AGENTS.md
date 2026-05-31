# Store Layer

Database storage using Ent with PostgreSQL. All database query/store types are defined in `store.go`.

## Structure

```
store/
├── store.go       # Adapter, connection management, all store types and DB queries
├── store_test.go  # Store unit tests
├── ent/           # Ent schema definitions and generated code
│   ├── schema/    # Ent schema definitions (tables) + domain types (types.go)
│   └── gen/       # Ent generated code
└── postgres/      # PostgreSQL adapter
```

## Rules

- All database query methods must be written in `store.go`. Do not create separate `xxx_store.go` files.
- Test files (`*_test.go`) are co-located with `store.go`.

## Patterns

- **Migrations**: Ent auto-migration via `client.Schema.Create()` on startup. No manual SQL migrations.
- Always use transactions for multi-step operations.
- All ORM operations through ent `gen.Client` (see `internal/store/postgres/adapter.go`).

## Commands

```bash
go tool task doc   # Generate schema docs
go tool task ent   # Generate ent code from schemas
```
