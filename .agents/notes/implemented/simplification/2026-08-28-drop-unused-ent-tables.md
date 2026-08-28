# Agent Note: Drop unused Ent tables

Status: implemented

## Problem

Six Ent schemas (`topics`, `urls`, `connections`, `authentications`, `platform_bots`, `capability_bindings`) were created early in the project but never wired to domain stores or production call paths. They add migration surface, generated code, and documentation drift without carrying business data.

## Decision

Remove the six schemas from `internal/store/ent/schema/`, regenerate Ent, delete related BDD specs, and update database reference docs. Deployments must manually `DROP TABLE` the physical tables — Ent `Schema.Create()` does not drop removed tables. See [deployment.md](../../docs/developer-guide/deployment.md#database-schema-upgrades).

| Removed table | Replacement |
| --- | --- |
| `topics` | string `topic` column on module tables (`oauth`, `data`, `configs`, …) |
| `urls` | none (abandoned design) |
| `connections` | `oauth` via `ModuleDataStore` |
| `authentications` | `oauth` via `ModuleDataStore` |
| `platform_bots` | `platforms` + `bots` + `platform_channels` via `PlatformStore` |
| `capability_bindings` | none (`HubStore` uses `apps` only) |

## Alternatives considered

- **Keep schemas for future use** — rejected; no open PRD or store design references these tables; pre-1.0 foundation-over-shims stance favors deletion.
- **Soft-deprecate only** — rejected; dead schema still generates code and confuses contributors.

## Consequences

- Breaking change for any external tool that queried these tables directly (none in-repo).
- Existing PostgreSQL instances retain empty/orphan tables until ops runs `DROP TABLE`.
- Ent `go generate` does not delete stale `gen/` files; remove orphans manually after schema deletion.

## Verification

```bash
go tool task ent
go tool task lint
go test ./internal/store/...
go tool task test:specs
```

Deploy SQL (when ready): see [deployment.md](../../docs/developer-guide/deployment.md#database-schema-upgrades).
