# Agent Note: Function sandbox workspace inject

Status: implemented

## Problem

Function Try / Invoke failed with `sh: 1: cannot open .flowbot-stdin: No such file` (exit 2) when Flowbot ran in a container and talked to the host Docker daemon via `docker.sock`. Ephemeral workspaces under `/tmp/flowbot-fn-*` were written inside the Flowbot container filesystem, then bind-mounted by path into the sandbox container. The daemon resolved that path on the **host**, which did not contain `main.go` / `.flowbot-stdin`, so the shell redirect failed before the entrypoint ran.

## Decision

Function sandbox Exec sets `WorkspaceInject`. On the Docker runtime, skip the workspace bind mount and copy the host workspace into `/workspace` via the Docker API (`CopyToContainer` tar) after create and before start. Tar entries are owned by uid/gid 1000 (image `agent`). Host env values that point under the ephemeral workspace (e.g. `GOCACHE` / `GOPATH`) are remapped to `/workspace/...`. Stdin redirect uses `/workspace/.flowbot-stdin`. Chat-agent sandbox keeps bind mounts (two-way host sync). When `Runtime` is kern, `WorkspaceInject` is ignored and the host workspace stays bind-mounted at the same path.

## Alternatives considered

- **Create function temps under `chat_agent.workspace`.** Rejected as the sole fix: a named volume mounted only into Flowbot is still invisible to the host Docker daemon for bind mounts; operators would need a host bind of the same path.
- **Docker Attach stdin instead of `.flowbot-stdin`.** Rejected as the sole fix: `main.go` / `go.mod` would still be missing under an empty bind.
- **Document shared `/tmp` bind only.** Rejected: easy to misconfigure; inject removes the footgun for ephemeral function runs.

## Consequences

- Function Try / Invoke works when Flowbot uses `docker.sock` from a container without sharing `/tmp` with the host.
- Inject path remapping and CopyToContainer are Docker-only; kern continues to bind-mount the host workspace at the same path.
- Docker inject WorkingDir is always `/workspace` (not the host temp path or a subdirectory).

## Verification

- `go test ./pkg/agent/sandbox/... ./internal/server/...`
- `TestEnvExecWorkspaceInject`, `TestEnvExecWorkspaceInjectIgnoredOnKern`, `TestTarWorkspace`, `TestRemapHostPathsInEnv`, `TestBuildHostConfig` inject cases
