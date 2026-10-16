# Agent Note: MCP per-token rate limit and audit query

Status: implemented

## Problem

Inbound MCP `tools/call` had no per-token budget and no argument-size cap. Audit rows could be written, but nothing could list them, and failure rows stored `err.Error()` plus a `failed` flag inside the request JSON.

## Decision

`tools/call` is limited by the access-token row id with a 60-second Redis sliding window. Every call counts toward `mcp.rate_limit.per_token_rpm` (default 120). Write and destructive tools also count toward `write_per_token_rpm` (default 20). `lineage.trace`, `lineage.search`, `flow.graph`, and `events.timeline` also count toward `lineage_per_token_rpm` (default 30). The first exhausted bucket rejects the call and writes nothing. Redis errors return retryable `unavailable`. `params.arguments` larger than `mcp.max_argument_bytes` (default 65536) return `invalid_argument` on field `arguments` and do not consume a bucket. Calls that pass the size check consume a bucket, including calls later rejected for scope.

`audit:read` lists rows whose `details.uid` equals the caller uid. `admin:*` can list every uid or filter one. Asking for another uid without `admin:*` returns `permission_denied`. The scope is not part of the MCP read preset. `GET /service/audit` and MCP `audit.list` share that rule. Groups `lineage`, `flows`, and `events` require `lineage:read`; group `audit` requires `audit:read`.

Mutation tools record `success`, `failure`, or `rejected`. Scope denial on a mutation records `rejected` / `permission_denied`. Rate-limit and oversize rejections are not stored. `error_code` is the tool error code. The list returns the sanitized request object. Past 4KiB the request stays an object with a `preview` string and `truncated: true`. Limit defaults to 20 and caps at 100. `correlation_id` stays in `details`. The page cursor is the row id. When more than one rate-limit bucket is full, `retry_after_ms` is the wait for the first full bucket.

`Mcp-Method` and `Mcp-Name` stay on the go-sdk check for protocol `2026-07-28`. This change does not duplicate that check.

`retention.flow_daily_days` zero means 365. The flow aggregate table itself is not built in this change.

## Alternatives considered

Keying the limiter by uid would let one agent exhaust every token of that user. A fixed `INCR` window can allow about twice the per-minute budget at the boundary. Failing open when Redis errors would drop the budget whenever the cache is down. Hashing the request would hide which entity was written. Filtering `result` from the JSON details in memory breaks the id cursor.

## Consequences

CLI callers of `/service/lineage`, `/service/flows`, and `/service/events` are not rate limited yet, because those handlers are not mounted. Web `kind=full` sessions stay on the global HTTP limiter only. Resource-link columns, trace, the flow aggregate, gosec exclude narrowing, and the flow page are not in this change. Correlation minting is recorded in [2026-10-17-data-event-correlation.md](2026-10-17-data-event-correlation.md). Existing audit rows keep an empty `result` column; new writes fill it.

## Verification

`go test ./internal/server/mcp/ ./internal/store/ -run TestAuditStore_ ./pkg/auth/ ./pkg/config/`
