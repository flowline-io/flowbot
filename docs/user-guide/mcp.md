# MCP

Flowbot speaks MCP in two directions. CLI and generated `docs/skills` stay as they are; MCP does not replace them.

## Inbound server (`/mcp`)

Set `mcp.enabled: true` and connect a client with a **header** token (`Authorization: Bearer` or `X-AccessToken`). Cookies and full web sessions (`kind=full`) are rejected. The endpoint is Streamable HTTP, stateless (MCP `2026-07-28`): POST is the protocol; GET and DELETE return 405. Default is off.

`GET /mcp/server-card` is a public [MCP Server Card](https://modelcontextprotocol.io/seps/2127-mcp-server-cards) (no token) when `mcp.enabled` is true and `mcp.server_card` is not false. The JSON is `$schema`, `name` (`flowline-io/flowbot`), `version`, `description`, and one `streamable-http` remote with protocol `2026-07-28`. The remote is `flowbot.url` plus `/mcp` when `flowbot.url` is set; otherwise it is the request host, and `X-Forwarded-Proto: https` or a TLS request selects https. It does not list tools. The extension schema is still pre-release, so those fields can change. `flowbot mcp config` prints that URL as the first stdout line, then the snippet. `-o json` puts it on `data.server_card`. The CLI prints the path for the endpoint you pass; it does not read this process's `mcp.server_card`. `/service/web/settings/mcp` shows the URL only while this process publishes the card.

Cursor example (`mcp.json`):

```json
{
  "mcpServers": {
    "flowbot": {
      "url": "http://127.0.0.1:6060/mcp",
      "headers": {
        "Authorization": "Bearer <access-token>"
      }
    }
  }
}
```

Use a narrow token (`service:karakeep:read`, `pipeline:run`, …) rather than `admin:*` when possible. `mcp.include` / `mcp.exclude` further crop the tool list (capability name or full `capability.operation`). A deny-list never exposes `core.run_terminal`, `core.run_code`, `core.http_request`, or `core.agent_run`. Write tools run when the token has write/run scope; they are not sent through chat-agent approval. Pipeline/workflow/function **apply** (YAML upsert) is not an MCP tool — use CLI or Web to change definitions; MCP can list, get, run, and inspect runs.

Successful calls return the payload in `structuredContent` and repeat that JSON as text. `pipeline.list`, `workflow.list`, `function.list`, and `hub.apps` use `items`. `get_run` without `run_id` returns `runs`. Other objects keep their fields. Every result includes `trace.request_id`, the same value as the HTTP `X-Request-ID` header. `trace.trace_id` is present when the call has a W3C trace.

Failed calls set `isError` and return `structuredContent.error` with `code`, `message`, `retryable`, and `hint`. The text body is the same JSON. Retry only when `retryable` is true, and wait `retry_after_ms` when that field is present.

### Idempotency keys

Non-read tools take an optional `idempotency_key` argument (`params.arguments.idempotency_key`, not `_meta`). Read tools do not advertise it and never create a claim. A key is 1 to 255 bytes of ASCII letters, digits, `.`, `_`, `~`, or `-`; the server removes it from the arguments before the handler runs.

A claim is a PostgreSQL row scoped to the access-token id, the tool name, and the key. Tool names are exact: the same key on `drone.deploy` and `kanboard.create_task` is two claims, and another token with the same key is a claim of its own. Claims do not expire: there is no 24-hour TTL, no Redis lock, and the unique constraint is the claim.

For an existing claim on the same token, tool, and key:

- Same arguments and a stored success: the handler does not run, the stored result is returned, and `trace.replayed` is true. `trace.correlation_id` is the first call's correlation id, so the replay does not emit a second data event.
- Same key with different arguments: `invalid_argument` with `field` set to `idempotency_key`, and the write does not run. Use a new key for a different payload.
- First call still running (`in_progress`): `unavailable` with `retry_after_ms` 1000. Retry the same call with the same key.
- Outcome `unknown`: `idempotency_unknown` with `retryable: false`. This covers a process that stopped mid-call, a handler failure after the claim, and a success body larger than `mcp.max_output_bytes`. An argument error releases the claim instead, so the same key runs again. Read with `*.list` or `*.get` before using a new key.

Mint one key before the first attempt and send that same key on every retry of that write. A different key is a new write, even with identical arguments, and a write without a key is not deduplicated. At startup, leftover `in_progress` claims become `unknown` and never execute again.

Key these first, because a duplicate side effect is hard to reverse: `drone.deploy` (rebuilds a deploy), `fireflyiii.create_transaction` (posts a second ledger entry), and `kanboard.create_task` (creates a second Kanban card). Every non-read tool accepts the key, including `pipeline.run`, `workflow.run`, and `function.run`.

The decision, hash canonicalization, and rejected alternatives (including a 12 to 24 hour TTL): [.agents/notes/implemented/feature/2026-10-21-write-idempotency-key.md](../../.agents/notes/implemented/feature/2026-10-21-write-idempotency-key.md).

Codes, hints, and schema scope: [.agents/notes/implemented/architecture/2026-10-16-mcp-tool-contract.md](../../.agents/notes/implemented/architecture/2026-10-16-mcp-tool-contract.md).

List tools take `limit` (default 20, maximum 100) and an opaque `cursor`. The next page is `page.next_cursor`. When a result is shortened to `mcp.max_output_bytes` (default 32768), `page.truncated` is true. `flowbot mcp config --client codex|claude|cursor|generic` prints a snippet and does not write client files. The Web page `/service/web/settings/mcp` shows the same snippets and can mint a read or run token for the signed-in user. `mcp.enabled` and `mcp.server_card` are read from the process config; changing them requires a restart.

## Client check

Confirm a client against a running instance. Record the client version and the result with the change that introduces the client; this file does not store that record.

1. Start an instance with `deployments/docker-compose.yaml`, `mcp.enabled: true`, and miniflux or karakeep test data.
2. On `/service/web/settings/mcp`, create a read-only token and `export FLOWBOT_MCP_TOKEN=...`.
3. Codex: paste the TOML, run `codex mcp list`, open `/mcp` in the TUI, and ask for the first 5 unread RSS entries. Confirm the page hint. A write call returns `permission_denied`.
4. Claude Code: run the printed `claude mcp add` command, confirm `/mcp`, and run the same read task.
5. Cursor: write `.cursor/mcp.json`, confirm the server and tools under Customize, run the same read task, and check MCP Logs.
6. Switch to a read-and-run token. In Codex set `default_tools_approval_mode = "writes"` and run `pipeline.run`. Codex should ask before the call.

## Outbound client (chat agent)

`chat_agent.mcp_servers` registers external MCP servers as agent tools named `mcp_<server>_<tool>`. HTTP uses `url` plus optional `headers`. stdio uses `command` + `args` (no shell, long-lived process). `npx` / `uvx` work but pull arbitrary code at start. There is no Web UI to add commands. URLs that would hit this process’s `/mcp` are rejected.

Permission key is `mcp.<server>` (default ask; read-only remote tools default allow). Auto-approval treats destructive/write MCP tools as flagged. DCG does not apply to external MCP tools.
