# Function sandbox workspace permissions

Status: implemented

## Problem

Function Try / Invoke ran `python main.py` in the agent sandbox and returned HTTP 422 with a generic validation message while logs showed `exit_code=2`. The ephemeral workspace under `/tmp/flowbot-fn-*` was created with default `0700` perms on a host user (often root in Docker). The sandbox container runs as uid 1000 and could not traverse the workspace or open `main.py`, so Python exited with code 2. Runtime invoke failures were also classified as `VALIDATION_ERROR` in the Try UI.

## Fix

- `sandbox.EnsureAgentReadable` prepares paths for uid 1000 (chown or world-accessible fallback).
- `functions.WorkspacePreparer` optional seam; `internal/server` `functionExecProvider` prepares invoke workspaces and agent-readables entrypoint writes / stdin files.
- Invoke runtime failures use `types.ErrInvokeRun` (non-zero exit, invalid stdout JSON, replayed failed runs). Web Try maps these to `INVOKE_FAILED` with `types.ClientMessage` detail; form/metadata validation stays `VALIDATION_ERROR`.
- `parseEntrypointResult` includes stderr/stdout in non-zero exit messages.
- When the Docker daemon cannot see Flowbot's `/tmp` (container + `docker.sock`), bind mounts alone are not enough; see [workspace inject](2026-09-04-function-sandbox-workspace-inject.md).

## Verification

- `go test ./pkg/functions/... ./pkg/agent/sandbox/... ./internal/modules/web/... ./internal/server/...`
- `TestEnsureAgentReadableOpensRestrictiveWorkspace`, `TestInvokeRunExitCodeAndStdoutJSON`, `TestFunctionWebTryInvokeRunFailed`
