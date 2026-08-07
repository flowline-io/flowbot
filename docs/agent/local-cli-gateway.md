# Local CLI Gateway

Flowbot can delegate coarse coding tasks to a **local** Cursor CLI process through `cmd/gateway` (`flowbot-gateway`).

This is **not** the [Notification Gateway](../user-guide/notification-gateway.md).

## Direction

Flowbot **never** dials the worker. The worker on your PC pulls jobs:

1. Chatagent tool `run_cursor` → CapGateway `run` creates a pending job in PostgreSQL.
2. `flowbot-gateway` heartbeats and claims jobs (`POST /gateway/v1/claim`).
3. Worker runs `agent -p --force --trust ...` (Cursor headless CLI) and posts the result.
4. CapGateway waits until the job is terminal (or times out / the turn is canceled).

## Server config (`flowbot.yaml`)

```yaml
gateway:
  enabled: true
  run_timeout: 30m
  worker_stale_after: 60s
  lease_ttl: 90s
  # permission: ask   # or allow — DefaultConfig only; user permission DB still overrides
```

No shared secret is stored in `flowbot.yaml`. Create a scoped access token:

```bash
go tool task run:composer -- admin token create
# select scope: gateway:worker
```

## Worker config

Copy [`cmd/gateway/gateway.yaml.example`](../../cmd/gateway/gateway.yaml.example) and set:

- `flowbot_url` — reachable Flowbot base URL
- `access_token` — token with `gateway:worker`
- `default_workspace` + `workspace_allowlist`
- optional `CURSOR_API_KEY` / `cursor_binary` (default `agent`)

Build / run:

```bash
go tool task build:gateway
./bin/flowbot-gateway -config gateway.yaml
./bin/flowbot-gateway -config gateway.yaml -log-level=debug   # claim empty / heartbeat detail
```

On startup the worker logs (via `flog`) whether `cursor_binary` resolves in `PATH`. Claim / run / complete / cancel and heartbeat failures are logged to stdout. Use `-log-level=debug` for empty-claim / heartbeat detail.

Release artifacts are named `flowbot-gateway_<os>_<arch>` (includes Windows).

## Security notes

- Cursor is invoked with `--force` (local file writes without interactive confirmation). Flowbot tool permission key `gateway` defaults to **ask**.
- Omit `cwd` in `run_cursor` to use the worker default workspace; non-empty paths must sit under the allowlist.
- Running jobs use a lease renewed by heartbeats; expired leases return to `pending` for another claim (also reclaimed on heartbeat / CapGateway wait).

## APIs (worker)

| Method | Path | Scope |
|--------|------|--------|
| POST | `/gateway/v1/claim` | `gateway:worker` |
| POST | `/gateway/v1/jobs/:id/result` | `gateway:worker` |
| POST | `/gateway/v1/heartbeat` | `gateway:worker` |
| GET | `/gateway/v1/jobs/:id` | `gateway:worker` |

Auth: `X-AccessToken` or `Authorization: Bearer` (same as CLI).
