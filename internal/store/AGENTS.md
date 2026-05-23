# Store Layer

Database storage using Ent with PostgreSQL.

## Structure

```
store/
├── store.go  # Adapter, connection management
├── model/    # Plain structs (DTOs)
├── ent/      # Ent schema definitions
├── ent/gen/  # Ent generated code
└── postgres/ # PostgreSQL adapter
```

## Patterns

- **Migrations**: Ent auto-migration via `client.Schema.Create()` on startup. No manual SQL migrations.
- Always use transactions for multi-step operations.
- All ORM operations through ent `gen.Client` (see `internal/store/postgres/adapter.go`).

## Commands

```bash
go tool task doc   # Generate schema docs
go tool task ent   # Generate ent code from schemas
```
