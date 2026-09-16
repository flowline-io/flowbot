# Agent Note: Sandbox CLI/credentials via Docker API inject

Status: implemented

## Problem

Chat-agent skill workflows that shell out to `flowbot` inside the Docker sandbox failed in production when Flowbot itself ran in a container with the host `docker.sock`. Operators had to set `chat_agent.sandbox.cli_path`, and even then temporary-dir bind mounts for `/opt/flowbot-cli` and `/home/agent/.config/flowbot` required the Docker daemon to see the same host paths the Flowbot process used. Missing CLI or empty config looked like a broken data plane while the real fault was inject path visibility.

## Decision

- Resolve the linux/amd64 CLI only as `flowbot-cli_linux_amd64` beside the server executable. Remove `chat_agent.sandbox.cli_path`.
- **Docker**: after container create, copy the CLI (named `flowbot`) and optional credential files into the sandbox via the Engine API (`CopyToContainer`). Do not bind-mount CLI or CLI config directories.
- **kern**: keep bind-based inject with the same sibling source and failure semantics.
- Always prepend `/opt/flowbot-cli` to `PATH` in the sandbox command.
- Failures warn and degrade: shell/code still run; a failing `flowbot` stub is injected when possible, otherwise `flowbot` is simply absent from the container.

Supersedes Docker bind staging described in [sandbox-cli-runtime-inject](../architecture/2026-08-17-sandbox-cli-runtime-inject.md) and [sandbox-cli-dir-bind](../bug-fix/2026-08-31-sandbox-cli-dir-bind.md) for the Docker runner. `server_url` must still match `listen` reachability ([sandbox-cli-server-url-port](../bug-fix/2026-09-16-sandbox-cli-server-url-port.md)).

## Alternatives considered

- **Keep `cli_path` as optional override** — rejected for this change; operators wanted zero path config. Local `go run` must place the sibling beside a packaged server binary instead.
- **Re-bake CLI into the sandbox image** — rejected earlier; rebuild cost and version drift vs the server deploy.
- **API inject with bind fallback** — rejected; dual paths recreate the DiD debugging surface.
- **Fail every Exec when the sibling is missing** — rejected; blocks unrelated shell/code. Stub on `PATH` fails only when `flowbot` is invoked.

## Consequences

- Official server images work without sandbox CLI path config when `/opt/app/flowbot-cli_linux_amd64` is present.
- Flowbot-in-Docker + host docker.sock no longer needs a shared host path for CLI/credentials.
- `go run` without a sibling CLI gets a stub when inject succeeds, otherwise `flowbot` is absent; place `flowbot-cli_linux_amd64` next to a real server binary for skill CLI workflows.
- Yaml that still sets `cli_path` is ignored by config unmarshal (field removed).

## Verification

- `go test ./pkg/agent/sandbox/...` covers sibling resolve, stub/config tar shape, kern stub/real staging, and Docker host config without CLI binds.
- Docs: [agent-sandbox.md](../../../../docs/agent/agent-sandbox.md) Chat agent CLI injection section.
