# Chat agent server default models (Web UI)

## Decision

Store server-wide chat agent model defaults in `configdata` (`uid=_server`, `topic=chatagent`, `key=server_defaults`) instead of writing `flowbot.yaml`. Runtime resolution: session override → DB server override (per field) → YAML → thinking fallback `default`.

## UI

`/service/web/chatagent-permissions` adds a **Server Default Models** block (chat_model, tool_model, thinking_level) with per-field inherit options and a separate bulk reset to YAML. User permission reset is unchanged.

## Access

Any authenticated Web user (same as existing permissions page).

## Out of scope

Settings catalog still shows YAML only; no new admin scope.

## Verification

```bash
go test ./internal/server/chatagent -run TestServerDefaults -count=1
go test ./internal/modules/web -run TestChatAgentPermissions -count=1
```

JSON permission saves do not touch server defaults (`submit_mode=json` skips `SaveServerDefaults`). Model pair validation lives in `config.ResolveChatAgentModelPair`.
