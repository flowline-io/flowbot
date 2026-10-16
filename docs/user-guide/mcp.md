# MCP

Flowbot speaks MCP in two directions. CLI and generated `docs/skills` stay as they are; MCP does not replace them.

## Inbound server (`/mcp`)

Set `mcp.enabled: true` and connect a client with a **header** token (`Authorization: Bearer` or `X-AccessToken`). Cookies and full web sessions (`kind=full`) are rejected. The endpoint is Streamable HTTP, stateless (MCP `2026-07-28`): POST is the protocol; GET and DELETE return 405. Default is off.

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

Use a narrow token (`service:karakeep:read`, `pipeline:run`, …) rather than `admin:*` when possible. `mcp.include` / `mcp.exclude` further crop the tool list (capability name or full `capability.operation`). A deny-list never exposes `core.run_terminal`, `core.run_code`, `core.http_request`, `core.agent_run`, `gateway.run`, or `gateway.cancel`. Write tools run when the token has write/run scope; they are not sent through chat-agent approval. Pipeline/workflow/function **apply** (YAML upsert) is not an MCP tool — use CLI or Web to change definitions; MCP can list, get, run, and inspect runs.

Successful calls return the payload in `structuredContent` and repeat that JSON as text. `pipeline.list`, `workflow.list`, `function.list`, and `hub.apps` use `items`. `get_run` without `run_id` returns `runs`. Other objects keep their fields. Every result includes `trace.request_id`, the same value as the HTTP `X-Request-ID` header. `trace.trace_id` is present when the call has a W3C trace.

Failed calls set `isError` and return `structuredContent.error` with `code`, `message`, `retryable`, and `hint`. The text body is the same JSON. Retry only when `retryable` is true, and wait `retry_after_ms` when that field is present.

Codes, hints, and schema scope: [.agents/notes/implemented/architecture/2026-10-16-mcp-tool-contract.md](../../.agents/notes/implemented/architecture/2026-10-16-mcp-tool-contract.md).

## Outbound client (chat agent)

`chat_agent.mcp_servers` registers external MCP servers as agent tools named `mcp_<server>_<tool>`. HTTP uses `url` plus optional `headers`. stdio uses `command` + `args` (no shell, long-lived process). `npx` / `uvx` work but pull arbitrary code at start. There is no Web UI to add commands. URLs that would hit this process’s `/mcp` are rejected.

Permission key is `mcp.<server>` (default ask; read-only remote tools default allow). Auto-approval treats destructive/write MCP tools as flagged. DCG does not apply to external MCP tools.
