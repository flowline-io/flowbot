# Self-hosting Flowbot

Single-instance, single-admin Homelab deployment. Flowbot is not designed for multi-replica HA.

## Requirements

- PostgreSQL 16+
- Redis 6+ (URL must include a non-empty password)
- Docker (compose path) or a Linux host for binary/systemd

## Quick start (Docker Compose)

1. Copy the reference config and adjust secrets:

```bash
cp docs/reference/config.yaml deployments/flowbot.yaml
```

Minimal edits in `deployments/flowbot.yaml`:

```yaml
listen: ":6060"
postgres:
  dsn: postgres://flowbot:flowbot@postgres/flowbot?sslmode=disable
redis:
  url: redis://:${REDIS_PASSWORD}@redis:6379/0
modules:
  web:
    auth:
      enabled: true
      username: admin
      password: ${WEB_PASSWORD}   # migrated into DB on first boot; remove after
      encryption_key: ${FLOWBOT_WEB_AUTH_ENCRYPTION_KEY}
      cookie_secure: false   # set true behind HTTPS
```

`${REDIS_PASSWORD}`, `${WEB_PASSWORD}`, and `${FLOWBOT_WEB_AUTH_ENCRYPTION_KEY}` are expanded from the process environment (see compose `environment`).

2. Start the stack from `deployments/`:

```bash
cd deployments
docker compose up -d --build
```

3. Open `http://localhost:6060/service/web/login` (or `/service/web/setup` on a fresh DB). Complete TOTP enrollment. Do not expose the management port to the public internet before the first admin account exists.

Health probes:

| Path | Meaning |
|------|---------|
| `/livez` | Process is up (Docker `HEALTHCHECK`) |
| `/readyz` | PostgreSQL + Redis ping OK; returns `503` while shutting down |

## Binary / systemd

See [developer-guide/deployment.md](developer-guide/deployment.md). Prefer `cookie_secure: true` and TLS termination at the reverse proxy.

## Reverse proxy

### Caddy

```caddy
flowbot.example.com {
  reverse_proxy 127.0.0.1:6060
}
```

### nginx

```nginx
location / {
  proxy_pass http://127.0.0.1:6060;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto $scheme;
}
```

When Flowbot sits behind a trusted proxy, set in `flowbot.yaml`:

```yaml
flowbot:
  url: https://flowbot.example.com
http:
  tls_behind_proxy: true
  trusted_proxies:
    - 127.0.0.1/32
    - 10.0.0.0/8
modules:
  web:
    auth:
      cookie_secure: true
```

`trusted_proxies` must list the reverse-proxy IPs/CIDRs; only then are `X-Forwarded-For` values used for login rate limiting. Set `flowbot.url` to the public HTTPS origin.

## Security baseline checklist

- [ ] Prefer `modules.web.auth.encryption_key` via env; after first boot remove plaintext `password` from YAML (accounts live in PostgreSQL)
- [ ] Keep login brute-force protection enabled (`modules.web.auth.brute_force.enabled`, default on)
- [ ] Do not expose `/service/web/setup` on a public network before the first account exists
- [ ] Lost TOTP: use backup codes, or `composer admin auth reset-2fa --username … --yes` on the host
- [ ] `cookie_secure: true` when serving HTTPS
- [ ] `flowbot.url` is the public HTTPS origin (e.g. `https://flowbot.example.com`)
- [ ] Inject secrets via `${ENV}` rather than committing them to YAML
- [ ] Restrict `/metrics` (`metrics.bearer_token` or scoped access token)
- [ ] Single instance only — do not run multiple Flowbot replicas against one Redis consumer group without reviewing cron/scan duplication

## Backup and restore

PostgreSQL is the system of record (events, runs, audit, tokens). Redis is transport/cache.

```bash
# Backup
pg_dump "$DATABASE_URL" -Fc -f flowbot-$(date +%Y%m%d).dump

# Restore
pg_restore -d "$DATABASE_URL" --clean --if-exists flowbot-YYYYMMDD.dump
```

After restore, restart Flowbot. Rebuild Redis Streams from PostgreSQL outbox if needed (see [Recovery](developer-guide/recovery.md)).

## Upgrade

1. Read [CHANGELOG.md](../CHANGELOG.md) for Breaking notes.
2. Back up PostgreSQL.
3. Pull the new image/binary and restart.
4. Schema changes are applied via Ent `Schema.Create` on startup (additive). Destructive config keys are rejected with a migration hint at load time.

## Single instance

Cron, Homelab scan, and event consumers assume one writer process. Scale vertically; do not run multiple app replicas unless you accept duplicate work.
