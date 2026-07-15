# Entry Points

3 binaries serving distinct roles.

| Binary   | Main file          | Purpose                                      | DI  |
| -------- | ------------------ | -------------------------------------------- | --- |
| server   | `main.go`          | HTTP API server (Fiber v3)                   | fx  |
| composer | `composer/main.go` | Dev tooling (`admin`, `webdoc`, `skills`)    | —   |
| cli      | `cli/main.go`      | User-facing CLI (login, hub, pipeline, workflows, capability commands) | —   |

## Commands

```bash
go tool task build:cli           # → bin/flowbot-cli
go tool task build:composer      # → bin/composer
```
