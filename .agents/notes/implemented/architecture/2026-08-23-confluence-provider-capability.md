# Confluence Cloud provider and capability (PR 2)

## Context

Second half of the Trello + Confluence rollout. Reuses the layering established in [2026-08-23-trello-provider-capability.md](2026-08-23-trello-provider-capability.md).

## Decisions

- **Deployment**: Atlassian Confluence Cloud (`site_url` + `/wiki/rest/api`).
- **Auth v1**: Email + Atlassian API token (Basic auth); OAuth deferred.
- **Capability**: Provider-native ops under `hub.CapConfluence`; scopes `service:confluence:read/write`.
- **Content**: `storage` XHTML only for create/update.
- **Events**: `confluence.page.{created,updated,deleted}` on mutations; inbound webhook via query token + automation JSON (`event`, `page`, `space`).
- **Webhook registration**: Manual in Confluence Automation (no register op).
- **Defaults**: Optional `default_space_key` when ops omit `space_key`.

## Verification

```bash
go tool task lint
go test ./pkg/providers/confluence/... ./pkg/capability/confluence/... -count=1
go tool task skills
```
