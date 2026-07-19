# Changelog

## Unreleased

### Breaking

- **Config: database** — replace `store_config` with top-level `postgres` (required `dsn`; optional pool / `max_results` / `sql_timeout` fields move next to `dsn`). Multi-adapter `use_adapter` / `adapters` map removed.
- **Config: Redis** — replace `redis.host` / `port` / `db` / `password` with required `redis.url` (password must be non-empty). Optional pool fields unchanged.
- Legacy keys are **rejected at load** (no silent ignore, no dual-read). See [config-reference.md](docs/reference/config-reference.md) migration table.
- **Web login brute force** — omitting `modules.web.auth.brute_force` now **enables** protection (was disabled). Set `brute_force.enabled: false` to disable.

### Added

- Official `deployments/docker-compose.yaml` (PostgreSQL + Redis + Flowbot) and [self-hosting guide](docs/self-hosting.md).
- Config `${ENV}` expansion for secrets; `http.trusted_proxies` for X-Forwarded-For trust.
- `/readyz` probes PostgreSQL + Redis and fails during shutdown.
- Optional `retention.data_events_days` cleanup (cascades related pipeline/outbox history); CLI aliases `karakeep`/`miniflux`/`kanboard`/`gitea`.
- Prebuilt Tailwind/DaisyUI CSS committed as `public/css/app.css` (no in-repo npm/`node_modules`); Alpine CSP build (no CSP `unsafe-eval`).

### Changed

- API error responses no longer leak internal `err.Error()` details; domain `types.Error` messages are preserved.
- OpenAPI `info` describes Homelab Data Hub (partial Swagger coverage documented).
- Media `max_size` / `gc_period` / `gc_block_size` default when zero (100 MiB / 60s / 100).
- Reference `docs/reference/config.yaml` shortened for infra + modules.web; `platform` / `vendors` stubs unchanged for now.
- Notify capability no longer advertises unimplemented `digest` op (use aggregate rules).
- Karakeep `delete` archives; Miniflux star/unstar via API.
