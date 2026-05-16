# Store Layer

Database storage using GORM with PostgreSQL.

## Structure

```
store/
├── store.go  # Adapter, connection management
├── model/    # GORM models (generated from DB schema)
└── mysql/    # PostgreSQL adapter
```

## Patterns

- **Migrations**: `pkg/migrate/migrations/` — `<timestamp>_<name>.up.sql` / `.down.sql`. Auto-run on startup.
- Always use transactions for multi-step operations.

## Commands

```bash
go tool task doc   # Generate schema docs
```
