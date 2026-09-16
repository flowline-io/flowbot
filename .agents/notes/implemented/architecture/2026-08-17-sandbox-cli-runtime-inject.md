# Agent Note: Sandbox CLI runtime inject

Status: implemented

## Problem

Giving chatagent access to provider capabilities via skill → `run_terminal` → sandbox `flowbot` CLI requires shipping a CLI release and rebuilding the `flowbot-agent-sandbox` image whenever the CLI changes. That packaging loop dominates development and deploy cost even when only the CLI (not the sandbox toolchain) changes.

## Decision

Keep the skill → CLI contract. Do not bake the `flowbot` CLI into `flowbot-agent-sandbox`.

Ship `flowbot-cli_linux_amd64` beside the Flowbot server binary (server image: `/opt/app/`). At runtime, `pkg/agent/sandbox` injects that sibling as `flowbot` under `/opt/flowbot-cli` on `PATH`. Docker copies via the Engine API; kern stages a host directory and bind-mounts it. Missing sibling or inject failure warns and degrades (failing stub when possible; shell/code still run). Host-native `task build:cli` is unchanged; `task build:cli:linux` builds the inject artifact.

Docker DiD path details: [sandbox-cli-api-inject](../simplification/2026-09-16-sandbox-cli-api-inject.md). Kern bind staging: [sandbox-cli-dir-bind](../bug-fix/2026-08-31-sandbox-cli-dir-bind.md).

Contract details: [Agent Sandbox — Chat agent CLI injection](../../../../docs/agent/agent-sandbox.md#chat-agent-cli-injection-chat_agentsandbox).

## Alternatives considered

- **Keep baking CLI into sandbox images** — rejected; every CLI change forces a `sandbox-v*` rebuild.
- **Product tools / in-process `capability.Invoke` for all caps** — out of scope; changes the product model beyond deploy-cost reduction.
- **Download CLI from GitHub releases at runtime** — rejected; reintroduces a release-asset dependency and version drift vs the running server.
- **Fail closed when CLI is missing** — rejected in favor of warn-and-degrade so shell/code sandboxes remain usable without skill CLI workflows.

## Consequences

- CLI version tracks the server deploy, not `sandbox-v*` toolchain tags.
- Operators place a linux CLI beside the server binary (official image already ships the sibling).
- New `sandbox-v*` images omit `/usr/local/bin/flowbot`; inject is required for skill → CLI workflows.

## Verification

- `go test ./pkg/agent/sandbox/...` covers sibling resolve, Docker API inject helpers, kern staging, and missing-CLI stub degrade.
- [`docs/agent/agent-sandbox.md`](../../../../docs/agent/agent-sandbox.md) and [`deployments/Dockerfile`](../../../../deployments/Dockerfile) / [`deployments/agent-sandbox/Dockerfile`](../../../../deployments/agent-sandbox/Dockerfile) document and implement inject vs bake.
- Sandbox CI smoke checks toolchain without requiring `flowbot version` in the image.
