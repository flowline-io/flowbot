# Changelog

## Unreleased

## [0.99.0] - 2026-08-15

### Added

- **Life** — solo gamified productivity module: quests (evidence, adjudication, dismiss), goals/plan with area linking, inventory/equipment, skill tree, achievements, real-life rewards, stats dashboard, and pagination across lists.
- **Email** capability with SMTP send and IMAP read, plus input sanitization.
- **Settings** webservice and FieldDocs-driven config documentation.
- **Functions** — named FaaS management via CLI and Automate web UI; agent sandbox ships a hermetic Go toolchain (`GOTOOLCHAIN=local`) for offline `go run`.
- **flowbot-agent** for local headless coding; local CLI gateway for job processing.
- **Pipeline** management commands in the Flowbot CLI.
- **agenteval** — agent evaluation CLI/CI with scenario filters, multi-trial scorecards, harness suite, and HTML reports.
- Chat agent: session trajectory view, tool-result compaction, tool-loop detection, approval modes, collapsed skills in the context popover, and richer message rendering.
- In-app inbox with deferred notification escalation.
- Data-event persist/publish path and outbox redelivery (including catch-up) for unpublished events.
- Web authentication management UI and TOTP enroll/reset flows.
- SEO configuration and generated assets; Redis list / sorted-set helpers and cache-key coverage.
- Slack thread replies and markdown divider support.
- Health snapshot sync and richer `/healthz` caching.

### Changed

- Directory creation defaults to mode `0750` or less (gosec G301); documented sandbox exception only. See [.agents/notes/implemented/process/2026-08-14-directory-create-mode-0750.md](.agents/notes/implemented/process/2026-08-14-directory-create-mode-0750.md).
- Agent sandbox shell entrypoint files use mode `0600` (no execute bit; run via interpreter). See [.agents/notes/implemented/process/2026-08-15-shell-entrypoint-file-mode-0600.md](.agents/notes/implemented/process/2026-08-15-shell-entrypoint-file-mode-0600.md).
- Decouple `pkg` from `internal` store bindings; pipeline engine uses a run-store adapter.
- Chat agent max steps raised; platform registration handles sole web-account cases.
- Gateway default-action handling and main error paths tightened.
- BDD/CI: AWS ECR image mirrors, pull fallbacks for rate limits, and more reliable Docker-dependent tests.
- Agent Notes + AGENTS.md process docs; website link verification and HTML checks.

### Security

- TOTP continues to use HMAC-SHA1 for authenticator interoperability (documented decision). See [.agents/notes/implemented/process/2026-08-15-webauth-totp-hmac-sha1.md](.agents/notes/implemented/process/2026-08-15-webauth-totp-hmac-sha1.md).

### Fixed

- Web route-group counts, Agent LLM proxy / race-test timeouts, docs link targets (`master`), and assorted auth/CSRF/login test isolation issues.

## [0.98.3] - 2026-07-26

Notable changes through this tag that were previously listed under Unreleased:

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
- Prebuilt Tailwind/DaisyUI CSS committed as `public/css/app.css` (no in-repo npm/`node_modules`); Alpine CSP `@alpinejs/csp` 3.15.12 (expression-capable CSP build, no `unsafe-eval`).

### Changed

- API error responses no longer leak internal `err.Error()` details; domain `types.Error` messages are preserved.
- OpenAPI `info` describes Homelab Data Hub (partial Swagger coverage documented).
- Media `max_size` / `gc_period` / `gc_block_size` default when zero (100 MiB / 60s / 100).
- Reference `docs/reference/config.yaml` shortened for infra + modules.web; `platform` / `vendors` stubs unchanged for now.
- Notify capability no longer advertises unimplemented `digest` op (use aggregate rules).
- Karakeep `delete` archives; Miniflux star/unstar via API.

[Unreleased]: https://github.com/flowline-io/flowbot/compare/v0.99.0...HEAD
[0.99.0]: https://github.com/flowline-io/flowbot/compare/v0.98.3...v0.99.0
[0.98.3]: https://github.com/flowline-io/flowbot/releases/tag/v0.98.3
