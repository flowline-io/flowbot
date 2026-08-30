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

Stage the CLI as a host directory containing an executable named `flowbot` (temp dir beside the source file, fallback to default temp), bind-mount that directory read-only at `/opt/flowbot-cli`, and prepend `/opt/flowbot-cli` to `PATH` inside the container (`PATH=/opt/flowbot-cli:$PATH` so the image PATH is preserved). Do not bind a file onto `/usr/local/bin/flowbot`. Staging failure warns once and degrades like a missing CLI.

Inject vs bake remains [sandbox-cli-runtime-inject](../architecture/2026-08-17-sandbox-cli-runtime-inject.md). Staging uses copy (not hardlink) so `chown` to uid 1000 cannot change the original binary's owner.

## Alternatives considered

- **Keep the file bind; document overlay2 as an ops issue.** Rejected: any resolved CLI made *all* sandbox execs fail.
- **Mounts API Type=bind instead of Binds.** Rejected: runc still `MkdirAll` the dest; the type mismatch remains.
- **Bind-mount the CLI parent directory (`/opt/app`).** Rejected: would expose the server binary and config into the sandbox.
- **Hardlink into the staging dir.** Rejected: `chown` on a hardlink changes the source inode.
- **Set Docker `PATH=` to the sandbox image PATH plus inject dir.** Rejected: that replaces a custom `chat_agent.sandbox.image` PATH.

## Consequences

- Skill → `run_terminal` → `flowbot` works on overlay2 when the sibling CLI exists.
- Each Exec copies the CLI into an ephemeral temp dir beside the source when the parent is writable.
- Custom sandbox images keep their own `PATH`; inject prepends `/opt/flowbot-cli`.
- A present CLI that cannot be staged does not fail the rest of shell/code exec.

## Verification

- `go test ./pkg/agent/sandbox/` covers directory binds at `/opt/flowbot-cli`, copy-not-hardlink, PATH wrap with `$PATH`, staging beside the source, and degrade when staging fails.
- [`docs/agent/agent-sandbox.md`](../../../../docs/agent/agent-sandbox.md) documents `/opt/flowbot-cli` inject.
