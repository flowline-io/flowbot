# Store Layer

Database storage using GORM with MySQL.

## Structure

```
store/
├── store.go  # Adapter, connection management
├── dao/      # DAOs (generated + custom)
├── model/    # GORM models (generated from DB schema)
└── mysql/    # MySQL adapter
```

## Patterns

- **Generated code**: `dao/*.gen.go`, `model/*.gen.go` — never edit. Regenerate with `go tool task dao`.
- **Custom DAOs**: `<entity>_dao.go` (not `.gen.go`) alongside generated code.
- **Migrations**: `pkg/migrate/migrations/` — `<timestamp>_<name>.up.sql` / `.down.sql`. Auto-run on startup.
- Always use transactions for multi-step operations.

## Commands

```bash
go tool task dao   # Regenerate DAOs
go tool task doc   # Generate schema docs
```
