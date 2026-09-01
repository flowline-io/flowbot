# Agent Note: Agent copy unification (chat agent → agent)

Status: implemented

## Problem

User-facing copy still said "Chat Agent" / "chat agent" in settings group titles, toasts, disabled-state banners, and config field descriptions while navigation and permissions pages already used "Agent" / "智能体".

## Decision

Replace user-visible "Chat Agent" / "chat agent" with "Agent" / "智能体" in i18n catalogs, settings field docs (via `pkg/config` godoc), DM fallback errors, REST API disabled errors, capability error messages, and knowledge metadata availability errors. Keep YAML keys (`chat_agent`), routes (`/chatagent/*`), and code identifiers unchanged.

## Alternatives considered

- **Rename config key and routes.** Rejected: out of scope for copy-only unification; high blast radius pre-1.0 but not required for operator clarity.
- **Leave settings YAML group as "Chat Agent".** Rejected: inconsistent with nav "Agent Settings" / "智能体设置".

## Consequences

- `chat_agent.chat_model` remains in operator-facing config hints where it names the actual YAML key.
- Historical CHANGELOG entries are not rewritten.

## Verification

- `TestSettingsGroupTitle` expects `settings.group.chat_agent` → "Agent".
- `go generate ./pkg/config/...` refreshes `field_docs_gen.go` and `settings_desc.zh.toml`.
