# Entry Points

Binaries:

| Binary   | Main file            | Purpose                                      | DI  |
| -------- | -------------------- | -------------------------------------------- | --- |
| server   | `main.go`            | HTTP API server (Fiber v3)                   | fx  |
| composer | `composer/main.go`   | Dev tooling (`admin`, `webdoc`, `skills`, `agenteval`) | —   |
| cli      | `cli/main.go`        | User CLI (login, hub, pipeline, workflows, …) | —   |
| gateway  | `gateway/main.go`    | Local CLI sidecar (`flowbot-gateway`; claims CapGateway jobs) | —   |
| agent    | `agent/main.go`      | Headless coding CLI (`flowbot-agent`; local loop, LLM via `/agent/v1`) | —   |

## Testing / build

```bash
go tool task build:cli           # → bin/flowbot-cli
go tool task build:cli:linux     # → bin/flowbot-cli_linux_amd64 (sandbox inject)
go tool task build:composer      # → bin/composer
go tool task build:gateway       # → bin/flowbot-gateway
go tool task build:agent         # → bin/flowbot-agent
```
