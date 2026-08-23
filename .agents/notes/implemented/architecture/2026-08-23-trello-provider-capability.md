# Trello provider and capability (PR 1)

## Context

Add Trello.com cloud integration as the first half of the Trello + Confluence provider rollout. Confluence follows in a second PR reusing the same layering.

## Decisions

- **Deployment**: Trello.com REST API only (`api.trello.com`).
- **Auth v1**: Static `api_key` + `token` in `vendors.trello`; OAuth deferred.
- **Capability**: Provider-native ops (`list_boards`, `create_card`, …) under `hub.CapTrello`; scopes `service:trello:read/write`. No `kanban` domain alias.
- **Events**: `trello.card.{created,updated,moved,deleted}` on mutations and inbound webhooks.
- **Webhook**: Query `webhook_token`; optional `register_webhook` / `delete_webhook` ops; HEAD probe on `/webhook/provider/trello/events` for Trello registration.
- **Defaults**: Optional `default_board_id` in config when ops omit `board_id`.
- **Homelab discovery**: Skipped (SaaS).
- **Delivery**: Trello full stack first; Confluence second PR.

## Verification

```bash
go tool task lint
go test ./pkg/providers/trello/... ./pkg/capability/trello/... ./pkg/client/... ./cmd/cli/command/... -count=1
go tool task skills
```
