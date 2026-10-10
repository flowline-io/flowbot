# Homelab image update checks

Flowbot periodically compares each scanned compose service's local **RepoDigest** with the registry manifest digest for the same image reference. When they differ, it emits `homelab.image.update_available`. Pipelines can notify; Hub Update (permission default off) still applies the pull.

This is digest movement on the **current tag** (including implicit `:latest`). It does not rewrite compose files, chase newer semver tags, pull, or roll back.

Decision record: [.agents/notes/implemented/feature/2026-10-10-homelab-image-update-check.md](../../.agents/notes/implemented/feature/2026-10-10-homelab-image-update-check.md).

## When it runs

Enabled when `homelab.runtime.mode` is `docker_socket` or `ssh`. Default interval is `6h`. Set `homelab.image_check.interval` to a Go duration; `0`, `0s`, or a negative duration disables it. The first cycle waits one interval after process start.

Digest queries run on the compose host through the homelab Runtime (the same Docker credentials as `Pull`). `runtime.mode: none` skips the checker.

## What is watched

Every tagged, non-digest-pinned service in scanned apps (the app allowlist still applies). Opt out per service:

```yaml
services:
  db:
    image: postgres:16
    labels:
      flowbot.image.watch: "false"
```

Services with no local RepoDigest (never pulled, or image loaded without a registry digest) are skipped and do not emit. Inspect failures (401, 429, timeout, missing buildx) are logged per service and skipped for that cycle.

## Event

| Field | Value |
| --- | --- |
| `EventType` | `homelab.image.update_available` |
| `Source` | `homelab_image_check` |
| `App` | compose app name |
| `Capability` | `flowbot.capability` provider ID when present |
| `EntityID` | `{app}/{service}` |
| `IdempotencyKey` | `{app}/{service}/{remote_digest}` |
| `Data` | `image`, `tag`, `current_digest`, `remote_digest` |

The same remote digest is not emitted twice. There is no "caught up" event.

Official blueprint: `homelab_image_update_notify` (event → `core.notify_send`).
