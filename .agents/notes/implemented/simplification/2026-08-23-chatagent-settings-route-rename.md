# ChatAgent settings page route rename

Status: implemented

## Problem

The settings page lived at `/service/web/chatagent-permissions` while its scope already included server default models, so the path read narrower than the page.

## Decision

Rename the web settings page path from `/service/web/chatagent-permissions` to `/service/web/chatagent-settings`, including POST sub-routes (`/reset`, `/reset-server-defaults`). Internal handler and template names stay permission-focused; only the public URL changes. Page title / nav copy alignment is owned by [agent-nav-order-and-settings-copy](2026-09-01-agent-nav-order-and-settings-copy.md).

## Alternatives considered

- **Keep `/chatagent-permissions` and only rename the heading.** Rejected: bookmarks and nav hrefs would still say permissions for a broader settings surface.

## Consequences

- Operators and tests use `/service/web/chatagent-settings`.
- Handler symbols remain `chatagent_permissions_*`.

## Verification

- `go test ./internal/modules/web -run TestChatAgentPermissions -count=1`
