# Agent Note: Sandbox CLI server_url must match listen

Status: implemented

## Problem

Chat-agent skill workflows that shell out to `flowbot bookmark` (and other CLI verbs) failed inside the Docker sandbox: `flowbot: not found`, then `Connection refused` to `host.docker.internal:6060`. Credentials were injected (`FLOWBOT_TOKEN` / config files), but the CLI binary was missing and the configured API port did not match the running server (`listen: ":6200"`).

## Decision

Keep runtime inject of `flowbot-cli_linux_amd64` (or `chat_agent.sandbox.cli_path`) and require `chat_agent.sandbox.server_url` to reach the same listen address the host process binds, via `host.docker.internal` (Docker) or `127.0.0.1` (kern + host network).

For local `go run` (server binary under `/tmp/go-build...`), set `cli_path` to an absolute linux/amd64 CLI — sibling auto-discovery beside the temp executable will not find `flowbot-cli_linux_amd64`. Build with `go tool task build:cli:linux`.

## Alternatives considered

- **Bake CLI into the sandbox image** — rejected; already decided against in [sandbox-cli-runtime-inject](../architecture/2026-08-17-sandbox-cli-runtime-inject.md).
- **Auto-rewrite server_url from listen** — rejected; sandbox reachability is not always the host listen address (reverse proxy, bind address, Docker network).
- **Fail closed when CLI is missing** — rejected; warn-and-degrade remains for shell/code-only runs.

## Consequences

- Operators changing `listen` must update `chat_agent.sandbox.server_url` (and usually `flowbot.url`) in the same edit.
- Dev machines using `go run` need an explicit `cli_path` after `task build:cli:linux`.
- An outdated host CLI (for example v0.96) can report empty bookmark lists against a newer server; prefer the inject binary from the same tree.

## Verification

- From a sandbox-like container with host-gateway: `http://host.docker.internal:6200` returns HTTP 200; `:6060` does not connect when the server listens on 6200.
- With inject dir on `PATH` and materialized config: `flowbot bookmark list -o json` returns bookmark items.
- `go test ./pkg/agent/sandbox/...` covers resolve, binds, and missing-CLI degrade.
