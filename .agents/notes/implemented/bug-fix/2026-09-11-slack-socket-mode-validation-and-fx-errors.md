# Slack Socket Mode validation and silent fx Provide errors

## Decision

When `platform.slack.enabled` is true, validate only `app_token` and `bot_token`. Do not require `app_id`, `client_id`, `client_secret`, or `signing_secret` — the wired Slack driver uses Socket Mode with bot + app-level tokens only.

Also print config/fx Provide failures before `flog.Init`: `NewConfig` logs validation and reachability errors via the standard library logger, and `FxLogger` falls back to stderr when flog is not yet initialized (zero-value zerolog discards).

## Alternatives considered

- **Require all Slack OAuth/signing fields when enabled.** Rejected: blocks Socket Mode-only configs that already have working tokens; those fields are unused by `internal/platforms/slack`.
- **Disable Slack in local `flowbot.yaml` only.** Rejected as the sole fix: leaves the product mismatch and silent exit for everyone else.
- **Initialize flog before config Provide.** Rejected: log config itself comes from `flowbot.yaml`; ordering stays Provide config → Init log.

## Consequences

- Socket Mode Slack starts with tokens alone; OAuth install fields remain optional until a future HTTP/OAuth path needs them.
- Misconfigured Slack or unreachable Postgres/Redis prints a visible error instead of bare `exit status 1`.
- `docs/developer-guide/cursor-cloud.md` documents Slack’s `app_token` + `bot_token` requirement.

## Verification

- `go test ./pkg/config/ ./pkg/flog/ -count=1` (includes Slack tokens-only OK and FxLogger error paths)
- `go tool task run` (or `go run -tags swagger ./cmd`) past config load without silent exit on a tokens-only Slack enablement
