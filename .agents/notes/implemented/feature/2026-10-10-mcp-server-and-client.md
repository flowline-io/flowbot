# Agent Note: MCP server and client

Status: implemented

## Problem

Flowbot’s capability layer is the natural tool catalog for a homelab, but AI clients had to go through generated SKILL.md plus a local CLI. Homelab users also already run third-party MCP servers (for example Home Assistant) that the chat agent could not call without a new provider.

## Decision

Flowbot ships both directions in one change.

Inbound `/mcp` is Streamable HTTP, stateless (`2026-07-28`), behind `mcp.enabled` (default false). Auth is `Authorization: Bearer` or `X-AccessToken` only; cookies and `kind=full` web sessions are rejected. Tools are one capability operation each, plus pipeline/workflow/function `list` / `get` / `run` / `get_run` and hub `apps` / `health`. Token scopes filter `tools/list`. `mcp.include` / `mcp.exclude` crop further. A hard deny-list never exposes `core.run_terminal`, `core.run_code`, `core.http_request`, `core.agent_run`, `gateway.run`, or `gateway.cancel`. Writes use token scope plus MCP destructive hints; they do not enter chat-agent approval. YAML apply stays on CLI/Web. Mutations audit as `mcp.tool.call`. HTTP lives in `internal/server/mcp`, not a module.

Outbound MCP is `chat_agent.mcp_servers`: HTTP or stdio (YAML, no shell, no command UI). Tools register as `mcp_<server>_<tool>` with permission key `mcp.<name>`. DCG does not apply. URLs that would hit this process `/mcp` are rejected. Implementation is `internal/server/chatagent/tools/mcp` using `modelcontextprotocol/go-sdk`.

CLI and `docs/skills` remain a third AI path.

## Alternatives considered

- **Split into two milestones.** Rejected: the user required one PR covering both surfaces.
- **Reuse `Authorize` (cookies) on `/mcp`.** Rejected: web `admin:*` sessions would become a remote homelab API.
- **Block MCP writes on chat-agent approval.** Rejected: MCP clients are not the Flowbot UI; token scope is the contract.
- **1:1 tools including `core.run_terminal` / `agent_run`.** Rejected: token-as-auth would hand those primitives to any MCP client.
- **Facade `capability_invoke` instead of 1:1 tools.** Rejected: discovery of `karakeep.create` is the product.
- **`internal/modules/mcp`.** Rejected: `/mcp` is a protocol gateway like `/chatagent`, not `/service/{module}`.
- **Pipeline apply on `/mcp`.** Rejected: LLM-generated YAML plus token-as-auth is a durable backdoor.
- **stdio command form in the Web UI.** Rejected: that is RCE as the Flowbot user.
- **Agent connecting to its own `/mcp`.** Rejected: doubled auth semantics and recursion.

## Consequences

- Operators must mint a scoped API token and set `mcp.enabled: true` before Cursor can connect.
- `admin:*` still sees every non-deny-list tool; include/exclude is the operational control for catalog size.
- stdio MCP processes run as the Flowbot user for the process lifetime.

## Verification

- `go test ./pkg/mcp/`
- `go test ./pkg/config/ -run TestValidate_MCPServers`
- `go test ./internal/server/mcp/`
- `go test ./pkg/agent/permission/ -run TestMCP`
- `go test ./pkg/agent/approval/ -run TestEvaluateFlaggedMCP`
- `go test ./internal/server/chatagent/eval -count=1`
- BDD (Docker): `tests/specs/mcp_spec_test.go`, `tests/specs/agent_spec_test.go` (`Agent Eval Policy`)
- User guide: [mcp.md](../../../docs/user-guide/mcp.md)
