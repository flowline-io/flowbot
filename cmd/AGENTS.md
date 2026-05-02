# Entry Points Guide

3 binaries serving distinct roles.

## Structure

```
cmd/
├── main.go             # HTTP server (Fiber, fx)
├── composer/           # Dev tool CLI (cli/v3)
│   ├── main.go
│   └── action/         # Subcommands: dao, doc
└── cli/                # Admin CLI
```

## Binaries

| Binary   | Main file          | Purpose                         | DI  |
| -------- | ------------------ | ------------------------------- | --- |
| server   | `main.go`          | HTTP API server (Fiber v3)      | fx  |
| composer | `composer/main.go` | Dev tools (dao gen, schema doc) | —   |
| cli      | `cli/main.go`      | Admin CLI commands              | —   |

## Anti-Patterns

- **Never** edit generated files in `composer/action/` output
- **Never** bypass fx container — use `fx.Provide`/`fx.Invoke`
- **Always** run `go tool task build:cli` after entry point changes

## Commands

```bash
go tool task build:cli      # Admin CLI
```
