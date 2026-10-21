# Agent Note: Merge lineage module into web

Status: implemented

## Problem

`internal/modules/lineage` existed only to mount `GET /service/audit` and expose `ListForCaller` for MCP. Keeping a separate module for one JSON list endpoint added registration surface without a lineage read API.

## Decision

`GET /service/audit` lives in `internal/modules/web` (`audit_webservice.go`), still under group `audit` so the path and `audit:read` gate are unchanged. The HTTP handler and MCP `audit.list` both call `store.AuditStore.ListForCaller` (`AuditCaller` carries uid, admin, and the requested subject). That function lives in `internal/store` so the two callers share it without a `web` ↔ `mcp` import cycle. The `lineage` module package and its `fx.Invoke` registration are removed.

## Alternatives considered

Keeping a dedicated module until lineage trace ships would leave an empty shell after audit moved. Putting `ListForCaller` on the web package would force `internal/server/mcp` to import `web`, which already imports MCP settings helpers and would cycle. Folding the route into `/service/web/*` would change the wire path and the auth group mapping.

## Consequences

Audit HTTP mounts whenever the web module's `Webservice` runs (same as other web routes). E2E specs use `web.MountAuditForE2E`. Scope names `lineage:read` and MCP tool groups named `lineage` are unchanged. The uid filter itself stays in [2026-10-17-mcp-rate-limit-and-audit-list.md](../feature/2026-10-17-mcp-rate-limit-and-audit-list.md).

## Verification

`go test ./internal/store/ -run TestVisibleSubject ./internal/modules/web/ -run TestAllWebserviceRuleGroups ./internal/server/mcp/ ./internal/modules/ -count=1`
