# Agent Note: Scanopy provider for network topology discovery

Status: implemented

## Problem

Homelab operators using [Scanopy](https://scanopy.net/docs/api/) for network topology discovery had no Flowbot provider client. NetAlertX covers device lists, but Scanopy exposes richer org/network-scoped hosts, services, and daemons via a versioned REST API.

## Decision

Add `pkg/providers/scanopy` as an API-key provider (Bearer `scp_u_…`) against Scanopy `/api/v1` and `/api/version`, then wire read ops through the existing `devops` aggregator (same pattern as NetAlertX):

- Config: `providers.scanopy.endpoint`, `providers.scanopy.api_key`
- Provider: health/version, list networks/hosts/services/daemons, get host
- Capability ops: `scanopy_health`, `scanopy_version`, `scanopy_list_*`, `scanopy_get_host`
- HTTP under `/service/devops/scanopy/*` and CLI `flowbot devops scanopy …`
- List pagination uses opaque `cursor` + `limit` at the capability/HTTP layer; the adapter maps to Scanopy `offset`/`limit` and fills `PageInfo` (`has_more`, `next_cursor`, `total`)
- Provider `ListParams.Limit` is `*int` so `LimitOf(0)` can send Scanopy's "no limit"

Scanopy is pre-v1.0 and documents an unstable API; the client models the public envelope (`success`/`data`/`meta`) and a focused read surface rather than the full CRUD matrix.

## Alternatives considered

- Standalone `scanopy` CapType — rejected; topology/device backends already live under `devops` with NetAlertX.
- Daemon API keys (`scp_d_…`) — rejected for this provider; docs recommend User API keys for programmatic integrations.
- Exposing raw `offset` on OpDef/HTTP — rejected; capability pagination must hide provider internals behind opaque cursors.

## Consequences

- Devops status/health include `scanopy` when configured.
- Provider types intentionally omit deep SCD2/topology-graph fields; expand when a consumer needs them.

## Verification

```bash
go test ./pkg/providers/scanopy/... ./pkg/capability/devops/... ./pkg/client/... ./cmd/cli/command/... ./internal/modules/hub/...
go tool task lint
```

Configure `providers.scanopy.endpoint` and `providers.scanopy.api_key`, then:

```bash
flowbot devops scanopy health
flowbot devops scanopy version
flowbot devops scanopy networks
flowbot devops scanopy hosts --search router
flowbot devops scanopy host --id <uuid>
flowbot devops scanopy services --network-id <uuid>
flowbot devops scanopy daemons
```
