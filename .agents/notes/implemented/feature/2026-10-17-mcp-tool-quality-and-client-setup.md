# Agent Note: MCP list pages, output cap, and client setup

Status: implemented

## Problem

MCP list tools did not share one page contract, and a single oversized result could exceed what a client model can read. Connecting Codex, Claude Code, and Cursor meant hand-writing each client's config and minting a token on the generic token form.

## Decision

List tools advertised on `/mcp` take `limit` (MCP fills 20 when omitted and clamps to 100) and an opaque `cursor`. Handlers that return a full slice without a page are sliced in `ApplyFullListPage`. `mcp.max_output_bytes` (default 32768) applies only when the MCP handler sets a context budget. Item-addressable `offset` cursors drop tail items and are re-signed; other strategies keep `next_cursor` and shorten strings, then replace an unsqueezable item with `{"_truncated":"item exceeds mcp.max_output_bytes","bytes":N}`. `page.truncated` is set when that happens. A cursor that cannot be decoded, or whose strategy is not an item offset, is not shortened by dropping rows; its `next_cursor` stays. The invoke cache stores the unclamped page, and a later call clamps only when its context has a budget. Unpaged slices keep the caller's `limit` when it is set, and use 50 when it is omitted.

`flowbot mcp config` prints the server card URL, then one snippet, on stdout. It does not write client files or token values. `/service/web/settings/mcp` shows the same snippets, the server card URL while the card is published, previews tools for a scope preset, and mints a token for the signed-in user. It does not change `mcp.enabled`. Server card: [note](2026-10-21-mcp-server-card.md). A tools/call for a known tool the token cannot use returns `permission_denied` without adding that tool to tools/list. The wizard scope lists live in `pkg/auth` because `pkg/config` already imports `pkg/mcp`, and `pkg/mcp` cannot import `pkg/auth`.

The checklist in `docs/user-guide/mcp.md` is unsigned. Its signing record is one row per client, with the version actually run, the date, and the result. That record is this table, not the user guide. Version, date, and result stay empty until a signing change fills them. A Claude Code run below 2.1.302 does not count. The unsigned checklist targets Codex 0.163.0. Cursor and Grok Build record the version from the day of the run. MCP Inspector 2.11 is a reference row and does not count as a client pass. An observed collapse of a tool to zero parameters is written in the same record. Input schema rules: [MCP input schema gate](../testing/2026-10-21-mcp-input-schema-gate.md).

| Client | Counts as | Version | Date | Result |
| --- | --- | --- | --- | --- |
| Claude Code | client; a run below 2.1.302 does not count | | | |
| Codex | client; the unsigned checklist targets 0.163.0 | | | |
| Cursor | client; version from the day of the run | | | |
| Grok Build | client; version from the day of the run | | | |
| MCP Inspector | reference at 2.11; not a client pass | | | |

## Alternatives considered

- Clamping only in `pkg/mcp` after `Invoke` returned. Rejected because re-signing an offset cursor needs the HMAC secret, which stays on the call context via `PublishCursorSecret`.
- Changing every adapter's default page size from 50 to 20. Rejected so HTTP and Pipeline callers that omit `limit` keep the existing default.
- A hot switch for `mcp.enabled`. Rejected because `/mcp` is mounted only at process start and settings are not written back to `flowbot.yaml`.
- Golden files for every tool description. Rejected. The lint checks length, `Example:`, parameter text, and declared enums. Wording is reviewed in the description diff.

## Consequences

Advertised tool descriptions follow a fixed shape: what it does, a tier sentence, then `Example:`. `pipeline:run`, `function:run`, and `workflow:run` in the run preset also authorize automate HTTP apply and delete. `service:core:read` and `service:life:read` are not in the wizard because they are not in `AllScopes`.

## Verification

`go test` for `pkg/capability`, `pkg/mcp`, `internal/server/mcp`, `cmd/cli/command`, and `internal/modules/web`. The checklist in `docs/user-guide/mcp.md` is unsigned. The signing record is the table in Decision.
