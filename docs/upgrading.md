# Upgrading

Operator checklist for breaking changes when moving between pre-1.0 builds. Chronological release notes stay in [CHANGELOG.md](../CHANGELOG.md). Orphan-table `DROP` statements that Ent will not remove: [Database schema upgrades](developer-guide/deployment.md#database-schema-upgrades). Config key renames: [config-reference.md](reference/config-reference.md#breaking-changes-infra-slim).

## Before every upgrade

1. Read this page and the target version's **Breaking** section in the changelog.
2. Back up PostgreSQL.
3. Deploy the new binary or image and restart.
4. Ent `Schema.Create` only adds or alters tables. Drop removed tables yourself when you no longer need the data.

## Tip of tree (Unreleased)

### Automate delete scopes

`pipeline:run`, `function:run`, and `workflow:run` still authorize apply and run. They no longer authorize `DELETE` under `/service/automate/...`. Delete needs `pipeline:delete`, `function:delete`, or `workflow:delete`. Mint a token with those scopes (Web MCP wizard: **Read, run, and delete**, or Custom), or use `admin:*`. MCP has no delete tool; CLI and REST deletes need the new scopes.

Rationale: [.agents/notes/implemented/architecture/2026-10-29-automate-delete-scope.md](../.agents/notes/implemented/architecture/2026-10-29-automate-delete-scope.md). Scope tables: [mcp.md](user-guide/mcp.md#scope-presets).

### `chat_agent.enabled`

Set `chat_agent.enabled: true` with a non-empty `chat_model`, or `enabled: false` to keep a model while the agent stays off. Omitting `enabled` while relying on a non-empty `chat_model` still turns the agent on, but startup logs a compat warning; set the flag explicitly before 1.0. When the agent is off, `core.agent_run` returns `unavailable`.

Rationale: [.agents/notes/implemented/feature/2026-11-06-chat-agent-enabled-switch.md](../.agents/notes/implemented/feature/2026-11-06-chat-agent-enabled-switch.md).

## 0.99.18

### Local CLI gateway and headless agent

`cmd/gateway`, `cmd/agent`, CapGateway, `run_cursor`, `/gateway/v1`, and `/agent/v1/chat/completions` are gone. Scopes `gateway:worker` and `agent:headless` authorize nothing. A leftover `gateway:` block in `flowbot.yaml` is ignored. Coding goes through the chat agent and its sandbox. Optional: drop leftover `gateway_jobs` / `gateway_workers` tables.

Rationale: [.agents/notes/implemented/simplification/2026-10-21-remove-local-cli-gateway-and-headless-agent.md](../.agents/notes/implemented/simplification/2026-10-21-remove-local-cli-gateway-and-headless-agent.md).

### Wasm and gRPC plugins

`pkg/plugin` and the `plugins` config section are gone. A leftover `plugins:` block is ignored. Extend Flowbot with in-tree modules, capabilities, and providers.

Rationale: [.agents/notes/implemented/simplification/2026-10-21-remove-wasm-grpc-plugin.md](../.agents/notes/implemented/simplification/2026-10-21-remove-wasm-grpc-plugin.md).

### Life

CapLife, `/service/web/life*`, and notify templates `life.quest.*` are gone. User-nav **Analytics** replaces them (Personal / Ops). Token usage is `/service/web/analytics/token-usage`; `/home/token-usage` redirects. Deployed databases keep `life_*` tables until you drop them.

Rationale: [.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md](../.agents/notes/implemented/feature/2026-10-21-analytics-replaces-life.md). Product surface: [analytics](analytics/README.md).

### Instruct and ghost configs

Desktop instruct pull/ack and the `instruct` table are gone. Host agents keep online, offline, and message. Drop `instruct` with the statements in [deployment.md](developer-guide/deployment.md#database-schema-upgrades). `platform.telegram` and vendor stubs `pocket`, `safeline`, and `wallos` leave the reference config; leftover keys are ignored.

Rationale: [.agents/notes/implemented/simplification/2026-10-21-remove-instruct-and-ghost-configs.md](../.agents/notes/implemented/simplification/2026-10-21-remove-instruct-and-ghost-configs.md).

### Removed uncalled providers

AdGuard Home, ArchiveBox, n8n, Slash, and Dropbox provider packages are gone (with `module.Shortcut` and Dropbox OAuth). Leftover `vendors` keys are ignored. Homelab discovery can still fingerprint ArchiveBox as capability `archive`.

Rationale: [.agents/notes/implemented/simplification/2026-10-21-remove-uncalled-providers.md](../.agents/notes/implemented/simplification/2026-10-21-remove-uncalled-providers.md).

## Older breakings operators still hit

| Area | Action |
| --- | --- |
| Config infra slim (`store_config` → `postgres`, Redis URL) | Load rejects legacy keys; see [config-reference](reference/config-reference.md#breaking-changes-infra-slim) |
| Web `/service/web` auth | Needs `admin:*` and `kind=full`; webhook `?token=` → `X-Webhook-Token` or HMAC ([0.99.13](../CHANGELOG.md)) |
| Automate module merge | Paths under `/service/automate/{functions\|pipeline\|workflow}`; `modules.automate.enabled` |
| MCP tool results (0.99.17) | JSON in `structuredContent` + text; protocol `2026-07-28` tools/call needs MCP headers — [mcp.md](user-guide/mcp.md) |

## See also

- [Self-hosting](self-hosting.md#upgrade)
- [CHANGELOG](../CHANGELOG.md)
- [MCP](user-guide/mcp.md)
