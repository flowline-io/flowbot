# Entry Points

3 binaries serving distinct roles.

| Binary   | Main file          | Purpose                    | DI  |
| -------- | ------------------ | -------------------------- | --- |
| server   | `main.go`          | HTTP API server (Fiber v3) | fx  |
| composer | `composer/main.go` | Dev tooling (admin, webdoc, skills, schema doc) | —   |
| cli      | `cli/main.go`      | Admin CLI commands                          | —   |
| chat     | `chat/main.go`     | Chat Agent terminal client (`flowbot-chat`) | —   |

## Commands

```bash
go tool task build:cli           # Admin CLI
go tool task build:chat          # Chat Agent TUI
go tool task build:composer      # Composer CLI
```
