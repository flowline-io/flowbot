# Entry Points

3 binaries serving distinct roles.

| Binary   | Main file          | Purpose                         | DI  |
| -------- | ------------------ | ------------------------------- | --- |
| server   | `main.go`          | HTTP API server (Fiber v3)      | fx  |
| composer | `composer/main.go` | Dev tools (dao gen, schema doc) | —   |
| cli      | `cli/main.go`      | Admin CLI commands              | —   |

## Commands

```bash
go tool task build:cli      # Admin CLI
```
