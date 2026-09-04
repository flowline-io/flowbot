# Agent Sandbox Image

`flowbot-agent-sandbox` is a Docker base image for **Cloud Agent ephemeral containers**. The orchestrator mounts a workspace, injects commands, and tears the container down when the agent run finishes. The image pre-installs git and common language toolchains so agents start ready to clone, build, and test code.

This image is **not** the Flowbot server container. See [Deployment](../developer-guide/deployment.md) for the main `flowbot` service image built from `deployments/Dockerfile`.

## How this differs from other sandboxes

| Mechanism | Scope | Purpose |
| --------- | ----- | ------- |
| **Agent sandbox image** (this document) | Ephemeral Docker container | Cloud Agent runtime environment |
| **Flowbot server image** (`deployments/Dockerfile`) | Long-running service | Run the Flowbot hub on port 6060 |
| **Chat agent workspace** (`chat_agent.workspace`) | Host directory path | File-system sandbox for DM chat agent tools |

The agent sandbox image follows the same principle as [Cursor Cloud Agent Dockerfiles](https://cursor.com/docs/cloud-agent/setup): **do not COPY the project into the image**. The orchestrator checks out or mounts sources at runtime under `/workspace`.

## Image variants

The Dockerfile defines two runtime stages:

| Stage | GHCR tag examples | Use when |
| ----- | ----------------- | -------- |
| `base` | `1.0.0`, `latest` | General coding agents: git, Go, Node, Python, shell tools |
| `playwright` | `playwright-1.0.0`, `playwright` | Browser automation or E2E tasks needing Chromium |

The Playwright variant adds roughly 400 MB (Chromium + system libraries). Pull it only when needed.

Registry: `ghcr.io/flowline-io/flowbot-agent-sandbox`

## Pre-installed toolchain

Versions are pinned in [`deployments/agent-sandbox/Dockerfile`](../../deployments/agent-sandbox/Dockerfile) build args and upgraded by releasing a new `sandbox-v*` tag.

| Tool | Version / source | Notes |
| ---- | ---------------- | ----- |
| Base OS | Ubuntu 24.04 | Required for Playwright and browser/computer-use tooling |
| git | distro package | Required for Cloud Agent clone workflows |
| sudo | NOPASSWD for `agent` | Privileged setup steps when orchestrator needs them |
| Go | 1.26.6 (official tarball) | Matches [`go.mod`](../../go.mod). `GOROOT=/usr/local/go`, `GOTOOLCHAIN=local` (no auto toolchain download). Used by Cloud Agents and by named FaaS (`go run main.go`, stdlib-only, `Network=none`) |
| Node.js | 22.x LTS (NodeSource) | Matches CI `node-version: lts/*` |
| oxfmt / oxlint | npm global | Matches Flowbot JS lint/format tooling |
| Python | 3.x (distro) | `python` symlinked to `python3`; pip and venv included |
| Shell / CLI | bash, jq, ripgrep, curl, wget, openssh-client, build-essential | Aligned with Flowbot server runtime packages |
| `dcg` | GitHub release (`DCG_VERSION`, default `v0.6.7`) | Installed as `/usr/local/bin/dcg` (linux musl amd64) with SHA256 verify; config at `/etc/dcg/config.toml` (same as [`pkg/agent/dcg/config.toml`](../../pkg/agent/dcg/config.toml)). **Parity only** — Flowbot's Always-on DCG gate for `run_terminal` / `run_code` runs on the **host** before sandbox exec; the image does not re-check. |

The image does **not** bake the `flowbot` CLI. Chat agent sandbox copies `flowbot-cli_linux_amd64` from beside the Flowbot server binary (or `chat_agent.sandbox.cli_path`) into a host directory and bind-mounts that directory at `/opt/flowbot-cli` (on `PATH`). See [Chat agent CLI injection](#chat-agent-cli-injection-chat_agentsandbox) below.

Credential files materialized by the chat agent sandbox runner are chowned to uid/gid `1000` (the image `agent` user) when possible; otherwise they are mode `0644` so the container can still read them when the host process cannot chown.

## Destructive Command Guard (dcg)

Flowbot chat agent Always-on protection for `run_terminal` / `run_code` uses [dcg](https://github.com/Dicklesworthstone/destructive_command_guard) on the **host** (`dcg --robot test` via `pkg/agent/dcg`), before permission ask and before sandbox exec:

- Install `dcg` on the Flowbot server `PATH` (required; missing binary → startup warning, first shell/code tool call fails closed).
- Policy is embedded from [`pkg/agent/dcg/config.toml`](../../pkg/agent/dcg/config.toml) (default cores + windows default packs + `remote` / `database` / `containers` / `system` / `platform`).
- `DCG_BYPASS` is stripped from the dcg child process only; agents have no bypass path.
- The sandbox image ships `dcg` + the same toml for **operational parity** only — Flowbot does not run a second check inside the container.

## Runtime contract

Ephemeral containers using this image should follow these conventions:

1. **Workspace** — Mount the agent working copy at `/workspace` (default `WORKDIR`).
2. **User** — Run as `agent` (uid/gid 1000) unless root is explicitly required.
3. **Command** — The image has **no ENTRYPOINT and no CMD**. The orchestrator supplies the process (e.g. shell, test runner, agent wrapper).
4. **No baked-in repo** — Do not rely on files copied at build time; clone or bind-mount at runtime.
5. **Lifecycle** — Use `--rm` (or equivalent) so containers are destroyed after each agent run.
6. **Flowbot CLI** — Not baked into the image. Chat agent sandbox injects the binary and credentials from `chat_agent.sandbox` (see below). Do not bake tokens into the image. Do not mount a human operator's host `~/.config/flowbot` in production.

Example:

```bash
docker run --rm \
  -u agent \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0 \
  bash -lc 'go test ./...'
```

Manual CLI against a host server (debug only; mount a linux/amd64 CLI binary):

```bash
docker run --rm \
  -u agent \
  -e FLOWBOT_SERVER_URL=http://host.docker.internal:6060 \
  -e FLOWBOT_TOKEN=your-access-token \
  --add-host=host.docker.internal:host-gateway \
  -v "$(pwd):/workspace" \
  -v /path/to/cli-dir:/opt/flowbot-cli:ro \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0 \
  bash -lc 'PATH=/opt/flowbot-cli:$PATH flowbot bookmark list'
```

`/path/to/cli-dir` is a directory that contains an executable named `flowbot`.

Older published CLI builds ignore `FLOWBOT_TOKEN` and only read `~/.config/flowbot/token`. Prefer mounting a materialized config directory (what Flowbot's sandbox runner does).

Playwright example:

```bash
docker run --rm \
  -u agent \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:playwright-1.0.0 \
  bash -lc 'npx playwright test'
```

## Registry and tags

| Git tag | Base image tags | Playwright image tags |
| ------- | --------------- | --------------------- |
| `sandbox-v1.0.0` | `1.0.0`, `1.0`, `sandbox-v1.0.0`, `latest` | `playwright-1.0.0`, `playwright-1.0`, `playwright-sandbox-v1.0.0`, `playwright` |
| `workflow_dispatch` + suffix `dev-abc` | `dev-abc` | `playwright-dev-abc` |
| `workflow_dispatch` (no suffix) | `dev-<sha>` | `playwright-dev-<sha>` |

## Versioning

Sandbox releases use **`sandbox-v*`** git tags, independent of Flowbot server releases (`v*`).

Release steps:

1. Merge Dockerfile or workflow changes to `main`.
2. Tag: `git tag sandbox-v1.0.0 && git push origin sandbox-v1.0.0`
3. GitHub Actions workflow [`docker-agent-sandbox.yml`](../../.github/workflows/docker-agent-sandbox.yml) builds and pushes both variants to GHCR.

Manual builds (development):

1. Open **Actions → Docker Agent Sandbox → Run workflow**.
2. Optionally set **tag suffix** (e.g. `dev-abc1234`).

## Build locally

```bash
# Slim base variant (context is repo root for dcg config COPY)
docker build -f deployments/agent-sandbox/Dockerfile --target base \
  -t flowbot-agent-sandbox:local .

# Playwright variant
docker build -f deployments/agent-sandbox/Dockerfile --target playwright \
  -t flowbot-agent-sandbox:playwright-local .

# Smoke test (no baked flowbot CLI)
docker run --rm flowbot-agent-sandbox:local bash -lc \
  'git --version && go version && node --version && python3 --version && dcg --version && test -f /etc/dcg/config.toml'

# FaaS offline Go contract (matches Network=none + GOPROXY=off)
docker run --rm --network=none -w /tmp \
  flowbot-agent-sandbox:local bash -lc \
  'printf "%s\n" "module flowbotfn" "" "go 1.26" > go.mod && printf "%s\n" "package main" "" "import \"fmt\"" "" "func main() { fmt.Println(\"{}\") }" > main.go && GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go run main.go'
```

## CI/CD

| Workflow | Trigger | Output |
| -------- | ------- | ------ |
| [`docker-agent-sandbox.yml`](../../.github/workflows/docker-agent-sandbox.yml) | Push tag `sandbox-v*`; manual `workflow_dispatch` | Pushes both `base` and `playwright` targets to GHCR |

Each matrix job runs a post-build smoke test (`git`, `go`, `node`, `python3`, `dcg`; offline `go run main.go` under `--network=none`; plus `playwright --version` for the Playwright variant) and prints `docker image inspect` Size ([packaging](../../.agents/notes/implemented/process/2026-08-30-agent-sandbox-image-packaging.md)).

## Orchestrator integration

Cloud Agent orchestrators should reference a pinned semver tag in production, for example:

- Default coding tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0`
- Browser / E2E tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:playwright-1.0.0`

### Named FaaS (`go run main.go`)

Functions invoke the sandbox with `Network=none` and no Flowbot CLI credentials. On Docker, the ephemeral host workspace is copied into `/workspace` (`WorkspaceInject`); see [.agents/notes/implemented/bug-fix/2026-09-04-function-sandbox-workspace-inject.md](../../.agents/notes/implemented/bug-fix/2026-09-04-function-sandbox-workspace-inject.md). Go entrypoints write a minimal `go.mod` (`module flowbotfn` / `go 1.26`) and run `go run main.go` with `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, and `CGO_ENABLED=0` (stdlib only; no module download). The image pins `GOTOOLCHAIN=local` so the bundled Go 1.26.6 compiler cannot attempt a toolchain fetch when the proxy is off.

### Chat agent CLI injection (`chat_agent.sandbox`)

When Flowbot runs shell/code tools in Docker sandbox mode, configure:

```yaml
chat_agent:
  sandbox:
    enabled: true
    image: ghcr.io/flowline-io/flowbot-agent-sandbox:latest
    server_url: "http://host.docker.internal:6060"
    access_token: "<hub-access-token>"
    # cli_path: ""  # optional; default is flowbot-cli_linux_amd64 beside the server binary
```

Behavior:

1. The runner resolves a host linux/amd64 CLI: `cli_path` if set (absolute as-is; relative beside the server executable), otherwise `flowbot-cli_linux_amd64` next to the Flowbot server executable (the server image ships this sibling under `/opt/app/`).
2. If that file exists, each Exec copies it into a temporary host directory as `flowbot` (beside the source file when possible), bind-mounts that directory read-only at `/opt/flowbot-cli`, and prepends `/opt/flowbot-cli` to `PATH` in the container command. If the file is missing or staging fails, the sandbox warns once and continues without `flowbot` (other shell/code tools still work). Mount shape: [.agents/notes/implemented/bug-fix/2026-08-31-sandbox-cli-dir-bind.md](../../.agents/notes/implemented/bug-fix/2026-08-31-sandbox-cli-dir-bind.md).
3. If `access_token` is non-empty, each Exec materializes a temporary host directory with `token` + `server_url` (mode `0600`), bind-mounts it read-only at `/home/agent/.config/flowbot`, and sets `FLOWBOT_TOKEN` / `FLOWBOT_SERVER_URL`.
4. The temp directory is outside the agent workspace and removed after the container exits.
5. If `server_url` host is `host.docker.internal`, the runner adds `ExtraHosts: host.docker.internal:host-gateway`.
6. Empty `access_token` skips credential injection (CLI calls fail with not logged in).

Local development on non-linux/amd64 hosts: build a linux CLI with `go tool task build:cli:linux` and either place `bin/flowbot-cli_linux_amd64` beside the running server binary or set `cli_path`.

### Kern runtime (`chat_agent.sandbox.runtime`)

Set `runtime: kern` to use the [kern](https://github.com/getkern/kern) CLI instead of Docker (Linux only; `kern` on PATH, `kern doctor` passing). Configure `security_profile: untrusted` for the hardened bundle. Use `server_url: http://127.0.0.1:6060` with `network: host` — `host.docker.internal` is not available under kern (Flowbot logs a warning). Workflow steps use the separate `kern:<image>` action; see [.agents/notes/implemented/architecture/2026-09-01-kern-executor-runtime.md](../../.agents/notes/implemented/architecture/2026-09-01-kern-executor-runtime.md).

Network options for `server_url`:

| Approach | When |
| -------- | ---- |
| `http://host.docker.internal:6060` + host-gateway | Default; Docker Desktop and Linux with host-gateway |
| `network: host` + `http://127.0.0.1:6060` | Linux same-host |
| Shared Docker network + service DNS | Flowbot and sandbox on the same user-defined network |

Release note: `FLOWBOT_TOKEN` requires a CLI build that supports it. Mounted config files keep older CLIs working.

Future Flowbot configuration for Cloud Agent runtime image selection will point at these GHCR coordinates.

## Extending the image

Fork or extend [`deployments/agent-sandbox/Dockerfile`](../../deployments/agent-sandbox/Dockerfile) when you need extra system packages or compiler versions. Keep stages separate so slim agents are not forced to pay for Playwright.

The Playwright stage installs **Chromium only** to limit image size. Add `firefox` or `webkit` in a custom stage if your orchestrator requires them.

## Limitations

The sandbox image intentionally does **not** include:

- Docker-in-Docker (large, requires privileged mode; Flowbot workflow Docker executor is separate)
- PostgreSQL or Redis (provide via orchestrator sidecars or host services)
- The Flowbot server binary or a baked `flowbot` CLI (chatagent injects `flowbot-cli_linux_amd64` at runtime)
- Pre-baked login tokens (inject at runtime via `chat_agent.sandbox.access_token` or manual Env/mount; never bake into the image)

## Related documentation

- [Agent Engine](./README.md) — `pkg/agent/` runtime
- [Deployment](../developer-guide/deployment.md) — Flowbot server and CI/CD overview
- [Architecture diagram](./agent-sandbox.puml) — CI → GHCR → ephemeral run flow
