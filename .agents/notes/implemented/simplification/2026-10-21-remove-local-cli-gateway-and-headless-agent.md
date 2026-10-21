# Agent Note: Remove local CLI gateway and headless agent

Status: implemented

## Problem

`cmd/gateway` and `cmd/agent` were a second coding path beside the chat agent: a pull worker, PostgreSQL job leases, CapGateway, the `run_cursor` tool, and an OpenAI-compatible LLM proxy. Homelab coding already goes through the chat agent and its sandbox. Keeping the worker binaries meant a parallel auth surface (`gateway:worker`, `agent:headless`), two release artifacts, and a capability that only those processes could complete.

## Decision

The local CLI gateway and the headless coding CLI are not part of Flowbot.

Removed with them: `cmd/gateway`, `cmd/agent`, CapGateway (`pkg/capability/gateway`), `/gateway/v1`, `/agent/v1/chat/completions`, `run_cursor`, gateway job and worker tables in the Ent schema, `gateway` config, scopes `gateway:worker` and `agent:headless`, and the headless coding-tool registrar. Chat-agent coding tools, the agent engine, the sandbox image `flowbot-agent-sandbox`, and the notification gateway stay.

Startup migration creates missing tables and does not drop tables that are absent from the schema. A database created before this removal may still contain `gateway_jobs` and `gateway_workers`. The config loader ignores a leftover `gateway:` block.

## Alternatives considered

- Delete only the two `cmd` trees and leave the HTTP APIs and CapGateway — rejected. Nothing in-tree can claim or finish those jobs, so the routes and the `run_cursor` tool would be a dead contract.
- Keep CapGateway disabled behind `gateway.enabled` for a later worker — rejected. Pre-release prefers one coding path over a dormant job queue, token scopes, and config.

## Consequences

- Release and task builds produce the server, composer, and CLI only.
- MCP no longer has a gateway deny-list entry, because that capability is not registered.
- Operators drop leftover `gateway_jobs` / `gateway_workers` themselves when they want those tables gone. Existing tokens that only had `gateway:worker` or `agent:headless` authorize nothing.

## Verification

- `go test ./pkg/capability/... ./pkg/agent/permission/... ./pkg/agent/tools/coding/... ./pkg/mcp/... ./internal/server/... ./internal/store/... ./cmd/composer/action/skills/...`
- `go tool task lint`
- Skill catalog and config field docs are regenerated from the remaining capability and config types.
