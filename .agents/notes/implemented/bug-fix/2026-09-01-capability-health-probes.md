# Agent Note: Capability health probes on /healthz

Status: implemented

## Problem

The `/service/web/healthz` capability table invokes `{cap}.health` for every registered capability. Several probes failed for code reasons rather than backend outages:

- `core` and `life` never registered a `health` operation (`not implemented`).
- `functions` returned `map[string]any{"healthy": bool}` while the healthz collector only accepted bare `bool` values.
- NetAlertX newer releases return 404 on deprecated `GET /devices/totals`; health and totals must use `GET /devices/totals/named` with legacy fallback.
- Memos misconfiguration often duplicated `/api/v1` in `providers.memos.endpoint`, producing 404 on `/api/v1/auth/me`.

## Decision

- Register aggregate `health` on `core` (always true when invoked) and `life` (`config.ChatAgentEnabled()`).
- Return `bool` from `functions.health`.
- NetAlertX: try `/devices/totals/named`, fall back to `/devices/totals` on 404.
- Memos provider: strip trailing `/api/v1` from configured endpoint before setting the Resty base URL.
- Healthz: also accept `map["healthy"]` so gateway-style responses stay compatible.

Provider auth and reachability issues (GitHub 401, Kanboard 403, Firefly III connection refused) remain configuration/runtime fixes outside this change.

## Verification

- `go test ./pkg/capability/core ./pkg/capability/life ./pkg/capability/functions ./pkg/providers/netalertx ./pkg/providers/memos ./internal/modules/web -run 'Test(RegisterExposesCoreOps|CapabilityHealthOK|NewMemos|NetAlertX)'`
