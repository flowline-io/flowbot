# Configuration Files

This directory contains configuration templates and examples for Flowbot.

## Breaking changes (infra slim)

| Old key | New key |
|---------|---------|
| `store_config.adapters.postgres.dsn` (+ pool fields under adapters.postgres) | `postgres.dsn` (+ same optional pool field names under `postgres`) |
| `store_config.use_adapter` / `store_config.adapters` | Removed (Postgres only) |
| `store_config.max_results` | `postgres.max_results` |
| `redis.host` + `redis.port` + `redis.db` + `redis.password` | `redis.url` (e.g. `redis://:PASSWORD@HOST:PORT/DB`) |

Loading **rejects** legacy keys (`store_config`, `redis.host` / `port` / `password` / `db`) with a migration hint. There is no dual-read compatibility.

### Environment substitution

String values in `flowbot.yaml` may use `${VAR}` / `$VAR` placeholders. They are expanded from the process environment at load (and on config reload). Prefer this for secrets in Docker Compose.

### Semantic change: web login brute force

Omitting `modules.web.auth.brute_force` used to disable lockout. It now **defaults to enabled**. Set `brute_force.enabled: false` to turn it off.

## Daily minimum

```yaml
listen: ":6060"
postgres:
  dsn: "postgres://flowbot:flowbot@localhost/flowbot?sslmode=disable"
redis:
  url: "redis://:flowbot@127.0.0.1:6379/0"
modules:
  - name: web
    enabled: true
    auth:
      username: admin
      password: "flowbot-dev-pass"
```

## File Descriptions

### `config.yaml`

Main application configuration template. The **opening sections** (listen, postgres, redis, modules.web) are the daily path. `platform` and `vendors` remain full stubs (future cleanup). Optional advanced knobs (`http`, `media`, pool overrides, `chat_agent`, `homelab`, …) are commented or documented below.

Covers:

- Server listen address and API path
- PostgreSQL (`postgres`)
- Redis (`redis.url` + optional pool)
- Logging, metrics, profiling, tracing
- Media storage (fs / MinIO) when enabled
- Executor, models, chat agent, homelab
- Platform integrations (Slack, Discord, Tailchat; Telegram struct only)
- Module settings (web auth)
- Third-party vendor stubs
- Capability invocation (`ability`)

Pipelines load from a separate `pipelines.yaml`. Notification templates/rules live in the UI / PostgreSQL (not in this file).

### Workflow Examples

See [examples/workflows/](../examples/workflows/) for workflow configuration examples.

## Quick Start

1. Copy the template:

   ```bash
   cp docs/reference/config.yaml flowbot.yaml
   ```

2. Set `postgres.dsn` and `redis.url` (non-empty password). Trim unused vendor/platform stubs as needed.

3. Start:

   ```bash
   task run
   ```

## Advanced field notes

| Area | Defaults when omitted |
|------|------------------------|
| `http.rate_limit` | max 200 / 10s |
| `postgres` pool / `sql_timeout` | adapter/pool.go defaults |
| `media.max_size` / `gc_period` / `gc_block_size` | 100 MiB / 60 / 100 |
| `modules.web.auth.cookie_secure` | true |
| `modules.web.auth.brute_force` | enabled; 5 / 10 / 15m / 15m |
| `metrics.enabled` | false (prefer false without a metrics backend) |

## Related

- [Homelab App Discovery](../user-guide/homelab-discovery.md)
- [Notification gateway](../user-guide/notification-gateway.md)
- [Database](database-reference.md)
- [CHANGELOG](../../CHANGELOG.md)
