# Agent Note: Sandbox CLI server_url must match listen

Status: implemented

## Problem

Chat-agent skill workflows that shell out to `flowbot bookmark` (and other CLI verbs) failed inside the Docker sandbox: `flowbot: not found`, then `Connection refused` to `host.docker.internal:6060`. Credentials were injected (`FLOWBOT_TOKEN` / config files), but the CLI binary was missing and the configured API port did not match the running server (`listen: ":6200"`).

## Decision

Keep runtime inject of `flowbot-cli_linux_amd64` beside the server binary and require `chat_agent.sandbox.server_url` to reach the same listen address the host process binds, via `host.docker.internal` (Docker) or `127.0.0.1` (kern + host network).

For local `go run` (server binary under `/tmp/go-build...`), place `flowbot-cli_linux_amd64` beside a packaged server binary — sibling auto-discovery beside the temp executable will not find it. Build with `go tool task build:cli:linux`. Docker inject uses the Engine API (see [sandbox-cli-api-inject](../simplification/2026-09-16-sandbox-cli-api-inject.md)).

## Alternatives considered

- **Bake CLI into the sandbox image** — rejected; already decided against in [sandbox-cli-runtime-inject](../architecture/2026-08-17-sandbox-cli-runtime-inject.md).
- **Auto-rewrite server_url from listen** — rejected; sandbox reachability is not always the host listen address (reverse proxy, bind address, Docker network).
- **Fail closed when CLI is missing** — rejected; warn-and-degrade remains for shell/code-only runs.

## Consequences

- Operators changing `listen` must update `chat_agent.sandbox.server_url` (and usually `flowbot.url`) in the same edit.
- Dev machines using `go run` need `flowbot-cli_linux_amd64` beside a packaged server binary after `task build:cli:linux`.
- An outdated host CLI (for example v0.96) can report empty bookmark lists against a newer server; prefer the inject binary from the same tree.

## Verification

- From a sandbox-like container with host-gateway: `http://host.docker.internal:6200` returns HTTP 200; `:6060` does not connect when the server listens on 6200.
- With sibling CLI inject and credentials: `flowbot bookmark list -o json` returns bookmark items.
- `go test ./pkg/agent/sandbox/...` covers sibling resolve, API/kern inject helpers, and missing-CLI stub degrade.
