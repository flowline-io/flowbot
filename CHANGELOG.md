# Changelog

## Unreleased

### Breaking

- **Config: database** — replace `store_config` with top-level `postgres` (required `dsn`; optional pool / `max_results` / `sql_timeout` fields move next to `dsn`). Multi-adapter `use_adapter` / `adapters` map removed.
- **Config: Redis** — replace `redis.host` / `port` / `db` / `password` with required `redis.url` (password must be non-empty). Optional pool fields unchanged.
- Legacy keys are **rejected at load** (no silent ignore, no dual-read). See [config-reference.md](docs/reference/config-reference.md) migration table.
- **Web login brute force** — omitting `modules.web.auth.brute_force` now **enables** protection (was disabled). Set `brute_force.enabled: false` to disable.

### Changed

- Media `max_size` / `gc_period` / `gc_block_size` default when zero (100 MiB / 60s / 100).
- Reference `docs/reference/config.yaml` shortened for infra + modules.web; `platform` / `vendors` stubs unchanged for now.
