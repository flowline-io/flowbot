# Cursor Cloud Environment

Single Go product (server on port `:6060`) plus CLI helpers under `cmd/`. Requires PostgreSQL + Redis. The update script only runs `go mod download`; everything below must be done per session because it is not part of the update script.

## Start services each session (systemd is unavailable)

```bash
sudo pg_ctlcluster 16 main start          # PostgreSQL 16 (data + role/db persist in snapshot)
sudo redis-server --daemonize yes --save "" --requirepass flowbot   # password MUST match flowbot.yaml
```

DB role/database are `flowbot`/`flowbot` (password `flowbot`, superuser). Recreate only if missing:

```bash
sudo -u postgres psql -c "CREATE ROLE flowbot LOGIN PASSWORD 'flowbot' SUPERUSER;" -c "CREATE DATABASE flowbot OWNER flowbot;"
```

Ent auto-migrates on server startup, so no manual migration step is needed.

## Config (`flowbot.yaml`, gitignored, already present at repo root)

Non-obvious validation gotchas (see `pkg/config/config.go` tags / `validate.go`) when deriving config from `docs/reference/config.yaml`:

- `redis.url` must include a non-empty password (e.g. `redis://:flowbot@127.0.0.1:6379/0`), so Redis is run with `--requirepass flowbot`.
- Platform `required_if=Enabled true` is **not** uniform: Discord requires app/client/bot credentials; Tailchat requires `api_url`. Slack and Telegram do **not** fail validation with empty tokens — still set unused platforms to `enabled: false` in Cloud.
- `GET /metrics` requires `metrics.bearer_token` or an access token with `admin:metrics` / `admin:*` scope.
- `/service/{capability}/*` (after Authorize) requires a minimum scope (`service:{capability}:read|write`, or `pipeline:*` for `/service/web/pipelines`, or `hub:capabilities:read` for `/service/hub`). Tokens with empty scopes are rejected. Web login still issues `admin:*`.
- `platform.tailchat.webhook_token` is required when Tailchat is enabled (header `X-Tailchat-Token`).
- `vendors.memos.webhook_token` is required for Memos webhooks (`?token=` query); empty config rejects deliveries like other providers.
- Prefer `metrics.enabled: false` when VictoriaMetrics is not running; leaving it on is harmless except push errors.
- `http.cors.allow_origins` defaults empty (no CORS reflection); `["*"]` never enables credentials. HSTS is sent when `http.tls_behind_proxy` or `modules.web.auth.cookie_secure` is true.
- Local DSN: `postgres.dsn` → `postgres://flowbot:flowbot@localhost/flowbot?sslmode=disable`.
- Redis: `redis.url` → `redis://:flowbot@127.0.0.1:6379/0` (password required in URL).
- Legacy keys `store_config` and `redis.host`/`port`/`password`/`db` are rejected at load with a migration hint.

## Run / build / lint / test

- Run dev server: `go tool task run` (uses `go run -tags swagger ./cmd`). Health: `/livez`, `/readyz`. Web UI: `/service/web/login` (creds from `modules.web.auth`; reference config uses `admin` / `flowbot-dev-pass`, or set `password_hash`).
- Lint (`go tool task lint`) includes a JS step (`oxlint ./public`); `oxlint` is installed globally via npm. If missing, run `npm install -g oxlint` (npm prefix must point inside the nvm node dir, e.g. `npm config set prefix "$HOME/.nvm/versions/node/v22.22.2"`, and that bin dir must be on PATH).
- Unit tests (`go tool task test`) pass without Docker and use the running Redis.
- `go tool task test:specs` (BDD) needs Docker/testcontainers, which is NOT installed here; install Docker first if you must run them. Without Docker, run unit tests and explicitly state that specs were not run.
