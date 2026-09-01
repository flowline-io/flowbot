# Kern executor runtime

## Context

Flowbot already runs workflow steps via `docker:` (Moby API) and chat-agent shell/code via an optional Docker sandbox. [kern](https://github.com/getkern/kern) is a daemonless Linux container runtime invoked through a single CLI binary.

## Decision

Add `kern` as an **optional** runtime alongside Docker:

- Workflow: `kern:<image>` action prefix → `pkg/executor/runtime/kern`
- Chat agent: `chat_agent.sandbox.runtime: docker | kern` (default `docker`)

Sandbox `DockerRunner` continues to call Moby directly; only `KernRunner` shares the kern CLI package with the workflow engine.

## Configuration ownership

| Surface | Config keys |
| ------- | ----------- |
| Workflow `kern:` steps | `executor.kern.*` only |
| Chat agent sandbox | `chat_agent.sandbox.*` only |

No cross-fallback between the two blocks.

## Semantics

- **Network (sandbox)**: empty / `none` → isolated box; `host` → `--net host` for same-host `127.0.0.1`. Other Docker network modes are rejected.
- **`host.docker.internal`**: when `runtime=kern`, startup warns to use `http://127.0.0.1:6060` + `network: host`.
- **Registry**: workflow tasks with `registry` credentials are rejected in v1; use public images or `kern login` on the host.
- **Bind mounts**: workflow bind mounts honor `executor.mounts.bind.allowed` (same policy as docker bind mounter).
- **Unsupported on kern workflow tasks**: GPU, overlay networks, volume/tmpfs mounts.

## Verification

```bash
go tool task lint
go test ./pkg/executor/runtime/kern/... ./pkg/workflow/... ./pkg/agent/sandbox/...
kern doctor && go test ./pkg/executor/runtime/kern/... -run Integration
```

## See also

- [docs/agent/agent-sandbox.md](../../../docs/agent/agent-sandbox.md) — Kern runtime subsection
- Plan: kern executor integration (workflow + sandbox, Linux native)
