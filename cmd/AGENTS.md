# Entry Points Guide

4 binaries serving distinct roles. All use `go.uber.org/fx` for dependency injection.

**Go Version:** 1.26

## Structure

```
cmd/
├── main.go             # HTTP server (Fiber, fx)
├── agent/              # Background daemon (cron, events, scripts)
│   ├── main.go         # Entry + reexec
│   ├── modules.go      # fx wiring
│   ├── daemon.go       # Core loop
│   └── config/ script/ startup/ ruleset/ client/ updater/
├── composer/           # Code generation CLI (cli/v3)
│   ├── main.go
│   └── action/         # Subcommands: migrate, generator, dao, doc
└── cli/                # Admin CLI
```

## Binaries

| Binary    | Main file          | Purpose                          | DI |
| --------- | ------------------ | -------------------------------- | -- |
| server    | `main.go`          | HTTP API server (Fiber v3)       | fx |
| agent     | `agent/main.go`    | Background daemon (cron, events) | fx |
| composer  | `composer/main.go` | Code gen, migrations, schema doc | —  |
| cli       | `cli/main.go`      | Admin CLI commands               | —  |

## Dependency Injection

Server and agent use `go.uber.org/fx` modules pattern:
```go
fx.New(server.Modules).Run()    // cmd/main.go
fx.New(Modules).Run()           // cmd/agent/main.go
```

Agent modules wiring (`cmd/agent/modules.go`):
```go
var Modules = fx.Options(
    fx.Provide(config.NewConfig, script.NewEngine, startup.NewStartup),
    fx.Invoke(RunDaemon, tickMetrics),
)
```

## Composer CLI

Code generation tool (`cli/v3`). Key subcommands:
```bash
composer migrate migration --name add_feature
composer generator bot --name mybot --rule command,form
composer generator vendor --name myvendor
composer dao --config ./flowbot.yaml
composer doc --config ./flowbot.yaml
```

## Agent Daemon

Background tasks: cron, events, script execution. `reexec` for self-upgrade.
```go
if reexec.Init() { return }
fx.New(Modules).Run()
```

## Anti-Patterns

- **Never** edit generated files in `composer/action/` output
- **Never** bypass fx container — use `fx.Provide`/`fx.Invoke`
- **Always** run `go tool task build:all` after entry point changes

## Commands

```bash
go tool task build          # Server binary
go tool task build:agent    # Agent daemon
go tool task build:composer # Composer CLI
go tool task build:app      # PWA admin
go tool task build:all      # All binaries
go tool task air            # Live reload (server only)
```
