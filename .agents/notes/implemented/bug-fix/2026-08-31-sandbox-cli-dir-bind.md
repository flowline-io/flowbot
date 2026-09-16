# Agent Note: Sandbox CLI directory bind

Status: implemented

## Problem

Chat-agent sandbox Exec failed at container start whenever a flowbot CLI file was present to inject:

```
error mounting "/opt/app/flowbot-cli_linux_amd64" to rootfs at "/usr/local/bin/flowbot":
create mountpoint for /usr/local/bin/flowbot mount: cannot create subdirectories in
".../merged/usr/local/bin/flowbot": not a directory
```

`pkg/agent/sandbox` used a Docker file bind (`hostFile:/usr/local/bin/flowbot:ro`). overlay2/runc `MkdirAll` the destination: if the image already has a file there the mkdir fails; if it does not, dest is created as a directory and a file-onto-directory mount fails. Either way every sandboxed command failed, not only `flowbot …`.

## Decision

Do not bind a file onto `/usr/local/bin/flowbot`. Inject a directory that contains an executable named `flowbot` at `/opt/flowbot-cli`, and prepend that dir to `PATH` (`PATH=/opt/flowbot-cli:$PATH`).

- **kern**: stage a host temp directory (copy of the sibling CLI, or a failing stub) and bind-mount it read-only.
- **Docker**: copy the same layout into the container via the Engine API instead of bind mounts — see [sandbox-cli-api-inject](../simplification/2026-09-16-sandbox-cli-api-inject.md).

Inject vs bake remains [sandbox-cli-runtime-inject](../architecture/2026-08-17-sandbox-cli-runtime-inject.md). Staging uses copy (not hardlink) so `chown` to uid 1000 cannot change the original binary's owner.

## Alternatives considered

- **Keep the file bind; document overlay2 as an ops issue.** Rejected: any resolved CLI made *all* sandbox execs fail.
- **Mounts API Type=bind instead of Binds.** Rejected: runc still `MkdirAll` the dest; the type mismatch remains.
- **Bind-mount the CLI parent directory (`/opt/app`).** Rejected: would expose the server binary and config into the sandbox.
- **Hardlink into the staging dir.** Rejected: `chown` on a hardlink changes the source inode.
- **Set Docker `PATH=` to the sandbox image PATH plus inject dir.** Rejected: that replaces a custom `chat_agent.sandbox.image` PATH.

## Consequences

- Skill → `run_terminal` → `flowbot` works when the sibling CLI injects successfully.
- Custom sandbox images keep their own `PATH`; inject prepends `/opt/flowbot-cli`.
- A present CLI that cannot be staged does not fail the rest of shell/code exec (stub or degrade).

## Verification

- `go test ./pkg/agent/sandbox/` covers `/opt/flowbot-cli` PATH wrap, kern copy-not-hardlink staging, stub degrade, and Docker host config without CLI binds.
- [`docs/agent/agent-sandbox.md`](../../../../docs/agent/agent-sandbox.md) documents `/opt/flowbot-cli` inject.
