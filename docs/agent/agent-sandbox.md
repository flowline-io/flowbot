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

The Dockerfile defines two stages:

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
| Go | 1.26.3 (official tarball) | Matches [`go.mod`](../../go.mod) |
| Node.js | 22.x LTS (NodeSource) | Matches CI `node-version: lts/*` |
| oxfmt / oxlint | npm global | Matches Flowbot JS lint/format tooling |
| Python | 3.x (distro) | `python` symlinked to `python3`; pip and venv included |
| Shell / CLI | bash, jq, ripgrep, curl, wget, openssh-client, build-essential | Aligned with Flowbot server runtime packages |

## Runtime contract

Ephemeral containers using this image should follow these conventions:

1. **Workspace** — Mount the agent working copy at `/workspace` (default `WORKDIR`).
2. **User** — Run as `agent` (uid/gid 1000) unless root is explicitly required.
3. **Command** — The image has **no ENTRYPOINT and no CMD**. The orchestrator supplies the process (e.g. shell, test runner, agent wrapper).
4. **No baked-in repo** — Do not rely on files copied at build time; clone or bind-mount at runtime.
5. **Lifecycle** — Use `--rm` (or equivalent) so containers are destroyed after each agent run.

Example:

```bash
docker run --rm \
  -u agent \
  -v "$(pwd):/workspace" \
  -w /workspace \
  ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0 \
  bash -lc 'go test ./...'
```

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
# Slim base variant
docker build -f deployments/agent-sandbox/Dockerfile --target base \
  -t flowbot-agent-sandbox:local deployments/agent-sandbox

# Playwright variant
docker build -f deployments/agent-sandbox/Dockerfile --target playwright \
  -t flowbot-agent-sandbox:playwright-local deployments/agent-sandbox

# Smoke test
docker run --rm flowbot-agent-sandbox:local bash -lc \
  'git --version && go version && node --version && python3 --version'
```

## CI/CD

| Workflow | Trigger | Output |
| -------- | ------- | ------ |
| [`docker-agent-sandbox.yml`](../../.github/workflows/docker-agent-sandbox.yml) | Push tag `sandbox-v*`; manual `workflow_dispatch` | Pushes both `base` and `playwright` targets to GHCR |

Each matrix job runs a post-build smoke test (`git`, `go`, `node`, `python3`; plus `playwright --version` for the Playwright variant).

## Orchestrator integration

Cloud Agent orchestrators should reference a pinned semver tag in production, for example:

- Default coding tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:1.0.0`
- Browser / E2E tasks: `ghcr.io/flowline-io/flowbot-agent-sandbox:playwright-1.0.0`

Future Flowbot configuration for Cloud Agent runtime image selection will point at these GHCR coordinates.

## Extending the image

Fork or extend [`deployments/agent-sandbox/Dockerfile`](../../deployments/agent-sandbox/Dockerfile) when you need extra system packages or compiler versions. Keep stages separate so slim agents are not forced to pay for Playwright.

The Playwright stage installs **Chromium only** to limit image size. Add `firefox` or `webkit` in a custom stage if your orchestrator requires them.

## Limitations

The sandbox image intentionally does **not** include:

- Docker-in-Docker (large, requires privileged mode; Flowbot workflow Docker executor is separate)
- PostgreSQL or Redis (provide via orchestrator sidecars or host services)
- The Flowbot server binary

## Related documentation

- [Agent Engine](./README.md) — `pkg/agent/` runtime
- [Deployment](../developer-guide/deployment.md) — Flowbot server and CI/CD overview
- [Architecture diagram](./agent-sandbox.puml) — CI → GHCR → ephemeral run flow
