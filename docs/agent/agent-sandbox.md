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

The Dockerfile defines a `cli-builder` stage plus two runtime stages:

| Stage | GHCR tag examples | Use when |
| ----- | ----------------- | -------- |
| `cli-builder` | (build only) | Downloads the `flowbot` CLI from GitHub releases |
| `base` | `1.0.0`, `latest` | General coding agents: git, Go, Node, Python, shell tools, `flowbot` CLI |
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
| Go | 1.26.3 (official tarball) | Matches [`go.mod`](../../go.mod) |
| Node.js | 22.x LTS (NodeSource) | Matches CI `node-version: lts/*` |
| oxfmt / oxlint | npm global | Matches Flowbot JS lint/format tooling |
| Python | 3.x (distro) | `python` symlinked to `python3`; pip and venv included |
| Shell / CLI | bash, jq, ripgrep, curl, wget, openssh-client, build-essential | Aligned with Flowbot server runtime packages |
| `flowbot` CLI | GitHub release (`CLI_VERSION`, default `latest`) | Installed as `/usr/local/bin/flowbot` from asset `flowbot-cli_linux_amd64`; checksum verified. This is the **CLI client** (second process vs host Flowbot server), not the server binary from `deployments/Dockerfile`. |

Credential files materialized by the chat agent sandbox runner are chowned to uid/gid `1000` (the image `agent` user) when possible; otherwise they are mode `0644` so the container can still read them when the host process cannot chown.

## Runtime contract

Ephemeral containers using this image should follow these conventions:

1. **Workspace** — Mount the agent working copy at `/workspace` (default `WORKDIR`).
2. **User** — Run as `agent` (uid/gid 1000) unless root is explicitly required.
3. **Command** — The image has **no ENTRYPOINT and no CMD**. The orchestrator supplies the process (e.g. shell, test runner, agent wrapper).
4. **No baked-in repo** — Do not rely on files copied at build time; clone or bind-mount at runtime.
5. **Lifecycle** — Use `--rm` (or equivalent) so containers are destroyed after each agent run.
6. **Flowbot CLI credentials** — Chat agent sandbox injects credentials from `chat_agent.sandbox.server_url` / `access_token` at runtime (temporary config mount + `FLOWBOT_*` env). Do not bake tokens into the image. Do not mount a human operator's host `~/.config/flowbot` in production.

Example:

```bash
docker run --rm \
  -u agent \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0 \
  bash -lc 'go test ./...'
```

Manual CLI against a host server (debug only):

```bash
docker run --rm \
  -u agent \
  -e FLOWBOT_SERVER_URL=http://host.docker.internal:6060 \
  -e FLOWBOT_TOKEN=your-access-token \
  --add-host=host.docker.internal:host-gateway \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0 \
  bash -lc 'flowbot bookmark list'
```

Older published CLI builds ignore `FLOWBOT_TOKEN` and only read `~/.config/flowbot/token`. Prefer mounting a materialized config directory (what Flowbot's sandbox runner does) until a CLI release with env support is baked into the sandbox image.

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
# Slim base variant (CLI from latest GitHub release tag)
docker build -f deployments/agent-sandbox/Dockerfile --target base \
  -t flowbot-agent-sandbox:local deployments/agent-sandbox

# Pin CLI to a release tag
docker build -f deployments/agent-sandbox/Dockerfile --target base \
  --build-arg CLI_VERSION=v0.40 \
  -t flowbot-agent-sandbox:local deployments/agent-sandbox

# Playwright variant
docker build -f deployments/agent-sandbox/Dockerfile --target playwright \
  -t flowbot-agent-sandbox:playwright-local deployments/agent-sandbox

# Smoke test
docker run --rm flowbot-agent-sandbox:local bash -lc \
  'git --version && go version && node --version && python3 --version && flowbot version'
```

## CI/CD

| Workflow | Trigger | Output |
| -------- | ------- | ------ |
| [`docker-agent-sandbox.yml`](../../.github/workflows/docker-agent-sandbox.yml) | Push tag `sandbox-v*`; manual `workflow_dispatch` | Pushes both `base` and `playwright` targets to GHCR |

Each matrix job runs a post-build smoke test (`git`, `go`, `node`, `python3`, `flowbot version`; plus `playwright --version` for the Playwright variant).

## Orchestrator integration

Cloud Agent orchestrators should reference a pinned semver tag in production, for example:

- Default coding tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0`
- Browser / E2E tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:playwright-1.0.0`

### Chat agent CLI credentials (`chat_agent.sandbox`)

When Flowbot runs shell/code tools in Docker sandbox mode, configure:

```yaml
chat_agent:
  sandbox:
    enabled: true
    image: ghcr.io/flowline-io/flowbot-agent-sandbox:latest
    server_url: "http://host.docker.internal:6060"
    access_token: "<hub-access-token>"
```

Behavior:

1. If `access_token` is non-empty, each Exec materializes a temporary host directory with `token` + `server_url` (mode `0600`), bind-mounts it read-only at `/home/agent/.config/flowbot`, and sets `FLOWBOT_TOKEN` / `FLOWBOT_SERVER_URL`.
2. The temp directory is outside the agent workspace and removed after the container exits.
3. If `server_url` host is `host.docker.internal`, the runner adds `ExtraHosts: host.docker.internal:host-gateway`.
4. Empty `access_token` skips credential injection (CLI calls fail with not logged in).

Network options for `server_url`:

| Approach | When |
| -------- | ---- |
| `http://host.docker.internal:6060` + host-gateway | Default; Docker Desktop and Linux with host-gateway |
| `network: host` + `http://127.0.0.1:6060` | Linux same-host |
| Shared Docker network + service DNS | Flowbot and sandbox on the same user-defined network |

Release note: `FLOWBOT_TOKEN` requires a CLI build that supports it. Until the sandbox image pulls that release, the mounted config files keep current published CLIs working.

Future Flowbot configuration for Cloud Agent runtime image selection will point at these GHCR coordinates.

## Extending the image

Fork or extend [`deployments/agent-sandbox/Dockerfile`](../../deployments/agent-sandbox/Dockerfile) when you need extra system packages or compiler versions. Keep stages separate so slim agents are not forced to pay for Playwright.

The Playwright stage installs **Chromium only** to limit image size. Add `firefox` or `webkit` in a custom stage if your orchestrator requires them.

## Limitations

The sandbox image intentionally does **not** include:

- Docker-in-Docker (large, requires privileged mode; Flowbot workflow Docker executor is separate)
- PostgreSQL or Redis (provide via orchestrator sidecars or host services)
- The Flowbot server binary (only the `flowbot` CLI client is pre-installed; point it at a server with `FLOWBOT_SERVER_URL` / mounted config)
- Pre-baked login tokens (inject at runtime via `chat_agent.sandbox.access_token` or manual Env/mount; never bake into the image)

## Related documentation

- [Agent Engine](./README.md) — `pkg/agent/` runtime
- [Deployment](../developer-guide/deployment.md) — Flowbot server and CI/CD overview
- [Architecture diagram](./agent-sandbox.puml) — CI → GHCR → ephemeral run flow
