# Entry Points

Binaries:

| Binary   | Main file            | Purpose                                      | DI  |
| -------- | -------------------- | -------------------------------------------- | --- |
| server   | `main.go`            | HTTP API server (Fiber v3)                   | fx  |
| composer | `composer/main.go`   | Dev tooling (`admin`, `webdoc`, `skills`, `agenteval`) | —   |
| cli      | `cli/main.go`        | User CLI (login, hub, pipeline, workflows, …) | —   |

## Testing / build

```bash
go tool task build:cli           # → bin/flowbot-cli
go tool task build:cli:linux     # → bin/flowbot-cli_linux_amd64 (sandbox inject)
go tool task build:composer      # → bin/composer
```
