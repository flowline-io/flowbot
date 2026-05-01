# Entry Points Guide

3 binaries serving distinct roles.

**Go Version:** 1.26

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

| Binary    | Main file          | Purpose                          | DI |
| --------- | ------------------ | -------------------------------- | -- |
| server    | `main.go`          | HTTP API server (Fiber v3)       | fx |
| composer  | `composer/main.go` | Dev tools (dao gen, schema doc)  | —  |
| cli       | `cli/main.go`      | Admin CLI commands               | —  |

## Dependency Injection

Server uses `go.uber.org/fx` modules pattern:
```go
fx.New(server.Modules).Run()    // cmd/main.go
```

## Composer CLI

Dev tools. Key subcommands:
```bash
composer dao --config ./flowbot.yaml
composer doc --config ./flowbot.yaml
```

## Anti-Patterns

- **Never** edit generated files in `composer/action/` output
- **Never** bypass fx container — use `fx.Provide`/`fx.Invoke`
- **Always** run `go tool task build:all` after entry point changes

## Commands

```bash
go tool task build          # Server binary
go tool task build:composer # Composer CLI
go tool task build:cli      # Admin CLI
go tool task build:all      # All binaries
go tool task air            # Live reload (server only)
```
