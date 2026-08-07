# Entry Points

Three binaries:

| Binary   | Main file            | Purpose                                      | DI  |
| -------- | -------------------- | -------------------------------------------- | --- |
| server   | `main.go`            | HTTP API server (Fiber v3)                   | fx  |
| composer | `composer/main.go`   | Dev tooling (`admin`, `webdoc`, `skills`, `agenteval`) | —   |
| cli      | `cli/main.go`        | User CLI (login, hub, pipeline, workflows, …) | —   |
| gateway  | `gateway/main.go`    | Local CLI sidecar (`flowbot-gateway`; claims CapGateway jobs) | —   |

## Testing / build

```bash
go tool task build:cli           # → bin/flowbot-cli
go tool task build:composer      # → bin/composer
go tool task build:gateway       # → bin/flowbot-gateway
```
