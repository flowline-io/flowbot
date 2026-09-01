# Changelog

## Unreleased

### Added

- **Scanopy** provider for network topology discovery (hosts, services, daemons) under the devops aggregator, with HTTP and CLI. See [.agents/notes/implemented/feature/2026-08-28-scanopy-provider.md](.agents/notes/implemented/feature/2026-08-28-scanopy-provider.md).
- Wakapi provider aligned with OpenAPI: WakaTime-compatible stats (`total_seconds`), health, all-time totals, and project search. See [.agents/notes/implemented/feature/2026-08-28-wakapi-provider-swagger.md](.agents/notes/implemented/feature/2026-08-28-wakapi-provider-swagger.md).
- Chat agent: microlighter syntax highlighting for markdown fences, and thread header meta chips (session, model, thinking, workspace). See [.agents/notes/implemented/feature/2026-08-30-chatagent-microlighter-codeblocks.md](.agents/notes/implemented/feature/2026-08-30-chatagent-microlighter-codeblocks.md).

### Changed

- User-facing copy unifies "Chat Agent" / "chat agent" to "Agent" / "智能体" in navigation, settings, toasts, and API errors. See [.agents/notes/implemented/simplification/2026-09-01-agent-copy-unification.md](.agents/notes/implemented/simplification/2026-09-01-agent-copy-unification.md).
- Navbar inbox/approval badges use one HTMX poller per endpoint; replica slots (mobile, Agents menu) update via OOB. See [.agents/notes/implemented/bug-fix/2026-08-30-nav-badge-single-poller.md](.agents/notes/implemented/bug-fix/2026-08-30-nav-badge-single-poller.md).

### Fixed

- Parallel LLM tool-call SSE is rewritten by `index` so OpenAI-compatible providers (e.g. MiMo) no longer reject multi-tool turns with empty arguments. See [.agents/notes/implemented/bug-fix/2026-08-30-llm-parallel-toolcall-stream-index.md](.agents/notes/implemented/bug-fix/2026-08-30-llm-parallel-toolcall-stream-index.md).
- LLM network errors are logged with a redacted URL and retried when no stream delta has been delivered. See [.agents/notes/implemented/bug-fix/2026-08-29-llm-network-error-sanitized.md](.agents/notes/implemented/bug-fix/2026-08-29-llm-network-error-sanitized.md).
- Fenced-code `language-*` class survives `MarkdownToSafeHTML` sanitization (labels and highlighting).
- Chat-agent approval EventSource closes on `pagehide` / same-tab navigation so inspect ↔ thread round-trips do not exhaust HTTP/1.1 sockets. See [.agents/notes/implemented/bug-fix/2026-08-30-chatagent-sse-pagehide-teardown.md](.agents/notes/implemented/bug-fix/2026-08-30-chatagent-sse-pagehide-teardown.md).
- Agent session inspect back link returns to the thread UI instead of the sessions list.
- Chat-agent sandbox injects the CLI via a directory bind at `/opt/flowbot-cli` instead of a file bind onto `/usr/local/bin/flowbot`, so overlay2/runc can start the container. See [.agents/notes/implemented/bug-fix/2026-08-31-sandbox-cli-dir-bind.md](.agents/notes/implemented/bug-fix/2026-08-31-sandbox-cli-dir-bind.md).

## [0.99.8] - 2026-08-28

### Added

- Manual retry of a failed pipeline run from the run detail page: same run ID, continue from the first failed step, replay successful step results. See [.agents/notes/implemented/feature/2026-08-28-pipeline-failed-run-retry.md](.agents/notes/implemented/feature/2026-08-28-pipeline-failed-run-retry.md).

### Changed

- Chat Agent permissions page path renamed to `/service/web/chatagent-settings`. See [.agents/notes/implemented/simplification/2026-08-23-chatagent-settings-route-rename.md](.agents/notes/implemented/simplification/2026-08-23-chatagent-settings-route-rename.md).
- Domain `Error()` includes `Cause` in operator logs and pipeline rows; HTTP JSON stays client-safe via `ClientMessage`. See [.agents/notes/implemented/bug-fix/2026-08-28-domain-error-includes-cause.md](.agents/notes/implemented/bug-fix/2026-08-28-domain-error-includes-cause.md).

### Removed

- Unused Ent tables `topics`, `urls`, `connections`, `authentications`, `platform_bots`, and `capability_bindings`. Existing databases keep the physical tables until ops runs `DROP TABLE`. See [.agents/notes/implemented/simplification/2026-08-28-drop-unused-ent-tables.md](.agents/notes/implemented/simplification/2026-08-28-drop-unused-ent-tables.md).

### Security

- SMTP send sanitizes untrusted MIME fields (CRLF/header injection, HTML UGC policy, quoted-printable bodies). See [.agents/notes/implemented/bug-fix/2026-08-23-email-content-injection.md](.agents/notes/implemented/bug-fix/2026-08-23-email-content-injection.md).

## [0.99.7] - 2026-08-23

### Added

- Chat agent server-wide default models in `configdata` (session override → DB → YAML). See [.agents/notes/implemented/feature/2026-08-23-chatagent-server-default-models.md](.agents/notes/implemented/feature/2026-08-23-chatagent-server-default-models.md).
- Hub `web` command returns a link to the Web UI.

### Fixed

- Layout CSS is cache-busted with `version.Buildtags`; session-badge SVG has intrinsic size so production navbar no longer inflates after deploy. See [.agents/notes/implemented/bug-fix/2026-08-23-navbar-prod-css-cache.md](.agents/notes/implemented/bug-fix/2026-08-23-navbar-prod-css-cache.md).
- Client i18n templates no longer render `<no value>` in chat-agent duration labels.

## [0.99.6] - 2026-08-23

### Changed

- Hub `deploy` notifies through the operator default template instead of unseeded `github.deployment`. See [.agents/notes/implemented/simplification/2026-08-23-deploy-default-notify-template.md](.agents/notes/implemented/simplification/2026-08-23-deploy-default-notify-template.md).

### Fixed

- Login POST 403 behind a TLS-terminating proxy: CSRF Origin rewritten when Fiber sees plaintext HTTP. See [.agents/notes/implemented/bug-fix/2026-08-23-csrf-login-403-tls-proxy.md](.agents/notes/implemented/bug-fix/2026-08-23-csrf-login-403-tls-proxy.md).

## [0.99.5] - 2026-08-23

### Added

- **Trello** provider and capability (`hub.CapTrello`): boards/cards, webhooks, and `trello.card.*` events. See [.agents/notes/implemented/architecture/2026-08-23-trello-provider-capability.md](.agents/notes/implemented/architecture/2026-08-23-trello-provider-capability.md).
- **Confluence Cloud** provider and capability (`hub.CapConfluence`): pages/spaces, inbound webhooks, and `confluence.page.*` events. See [.agents/notes/implemented/architecture/2026-08-23-confluence-provider-capability.md](.agents/notes/implemented/architecture/2026-08-23-confluence-provider-capability.md).

## [0.99.4] - 2026-08-22

### Fixed

- Settings description locale generator writes into an explicit output directory.

## [0.99.3] - 2026-08-22

### Added

- Web UI i18n (en/zh) for `/service/web` and public clip pages via embedded go-i18n catalogs. UI locale is cookie-based and separate from `flowbot.language` (ChatAgent / LLM replies). See [.agents/notes/implemented/feature/2026-08-22-web-ui-i18n.md](.agents/notes/implemented/feature/2026-08-22-web-ui-i18n.md).

## [0.99.2] - 2026-08-22

### Added

- Chat agent composer workspace picker: lock a session to `chat_agent.workspace` or a first-level subdirectory. See [.agents/notes/implemented/feature/2026-08-17-chatagent-composer-workspace-picker.md](.agents/notes/implemented/feature/2026-08-17-chatagent-composer-workspace-picker.md).
- `web_search` falls back to keyless DuckDuckGo HTML search when `chat_agent.web_search.api_key` is unset. See [.agents/notes/implemented/feature/2026-08-22-web-search-duckduckgo-fallback.md](.agents/notes/implemented/feature/2026-08-22-web-search-duckduckgo-fallback.md).
- Shared confirmation modal for dangerous Web UI actions.

### Changed

- HTTP app mounts Fiber Helmet → CORS → CSRF; CSRF cookie is `__Host-csrf_` when `cookie_secure` is on. See [.agents/notes/implemented/architecture/2026-08-18-fiber-security-middleware-stack.md](.agents/notes/implemented/architecture/2026-08-18-fiber-security-middleware-stack.md).
- CI `actions/setup-go` pinned to Go 1.26.6. See [.agents/notes/implemented/process/2026-08-18-ci-setup-go-patch-pin.md](.agents/notes/implemented/process/2026-08-18-ci-setup-go-patch-pin.md).

### Security

- Helmet sets `X-Frame-Options: DENY`, HSTS `max-age=63072000; includeSubDomains` (no preload), and a camera/microphone/geolocation Permissions-Policy.

## [0.99.1] - 2026-08-17

### Added

- Sandbox CLI runtime inject: ship `flowbot-cli_linux_amd64` beside the server and bind-mount it into ephemeral sandbox containers (`sandbox-v0.6.0` images no longer bake `/usr/local/bin/flowbot`). See [.agents/notes/implemented/architecture/2026-08-17-sandbox-cli-runtime-inject.md](.agents/notes/implemented/architecture/2026-08-17-sandbox-cli-runtime-inject.md).
- Architecture documentation for the Flowbot runtime.

### Fixed

- Dockerfile entrypoint syntax; `TestResolveCLIBinary` race under parallel execution.

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
- Agent sandbox shell entrypoint files use mode `0600` (no execute bit; run via interpreter). See [.agents/notes/archived/process/2026-08-15-shell-entrypoint-file-mode-0600.md](.agents/notes/archived/process/2026-08-15-shell-entrypoint-file-mode-0600.md).
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

[Unreleased]: https://github.com/flowline-io/flowbot/compare/v0.99.8...HEAD
[0.99.8]: https://github.com/flowline-io/flowbot/compare/v0.99.7...v0.99.8
[0.99.7]: https://github.com/flowline-io/flowbot/compare/v0.99.6...v0.99.7
[0.99.6]: https://github.com/flowline-io/flowbot/compare/v0.99.5...v0.99.6
[0.99.5]: https://github.com/flowline-io/flowbot/compare/v0.99.4...v0.99.5
[0.99.4]: https://github.com/flowline-io/flowbot/compare/v0.99.3...v0.99.4
[0.99.3]: https://github.com/flowline-io/flowbot/compare/v0.99.2...v0.99.3
[0.99.2]: https://github.com/flowline-io/flowbot/compare/v0.99.1...v0.99.2
[0.99.1]: https://github.com/flowline-io/flowbot/compare/v0.99.0...v0.99.1
[0.99.0]: https://github.com/flowline-io/flowbot/compare/v0.98.3...v0.99.0
[0.98.3]: https://github.com/flowline-io/flowbot/releases/tag/v0.98.3
