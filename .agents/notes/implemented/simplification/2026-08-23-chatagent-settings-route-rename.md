# ChatAgent settings page route rename

## Decision

Rename the web settings page path from `/service/web/chatagent-permissions` to `/service/web/chatagent-settings`, including POST sub-routes (`/reset`, `/reset-server-defaults`). Page and nav copy now use "Chat Agent Settings" / "Agent Settings" (en) and "Chat Agent 设置" / "智能体设置" (zh). Internal handler and template names stay permission-focused; only the public URL and user-facing strings change.

## Rationale

The page now covers server default models and user permissions — "settings" matches scope better than "permissions".

## Verification

```bash
go test ./internal/modules/web -run TestChatAgentPermissions -count=1
```
