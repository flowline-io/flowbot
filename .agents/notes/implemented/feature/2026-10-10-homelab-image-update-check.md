# Agent Note: Homelab image update checks

Status: implemented

## Problem

Homelab compose apps pin a tag (often `:latest`) and operators learn about registry movement from a separate tool such as Diun. Flowbot already scans those apps and can `Pull`/`Update`, but it had no remote digest inspection and no domain event for "this running image is behind its tag".

## Decision

v1 inspects **digest movement on the existing image reference** and emits `homelab.image.update_available`. It does not rewrite compose files, chase newer tags, auto-upgrade, health-gate, or roll back.

Comparison is remote tag **manifest/index digest** versus the local **RepoDigest** whose repository name matches the compose image (running container, else local image). Missing or unmatched RepoDigests skip the service. Queries go through homelab Runtime on the compose host. `runtime.mode` `docker_socket` or `ssh` enables the loop; default interval `6h`, serial, first run after one interval. `homelab.image_check.interval` of `0`, `0s`, or a negative duration disables. Scanned tagged services are watched unless `flowbot.image.watch=false`. Per-service inspect failures log and skip. Dedup is an EventStore lookup of `event_type` + `idempotency_key` (non-unique index); no watch-state table. `DataEvent.Capability` is the `flowbot.capability` label, not probe fingerprints.

Official blueprint `homelab_image_update_notify` wires the event to `core.notify_send`. Hub Update remains the apply path.

## Alternatives considered

- **New semver tag discovery in v1.** Rejected: that mutates compose image references; `docker compose pull` does not.
- **Compare last-seen remote digest only (Diun).** Rejected: Flowbot knows running containers; "update available" means the live app is behind.
- **Approval Inbox, auto Update, health check, rollback in v1.** Rejected: Inbox has no approve action; Runtime Update has no rollback; fake safety is worse than detect-only.
- **In-process registry HTTP client.** Rejected: a second credential plane, and SSH mode would still need the compose host.
- **Dedicated watch-state table.** Rejected: v1 has no Hub pending list; `data_events` is the durable record.
- **Opt-in watch label.** Rejected: silent default; app allowlist already scopes apps.

## Consequences

- Operators need `docker buildx imagetools inspect` on the compose host for remote digests.
- Multi-arch tags are compared at the index digest, not the platform image id.
- Sidecar `:latest` services notify unless labelled out.
- Upgrade, approval, and rollback remain a later slice.

## Verification

- `go test ./pkg/homelab/ -run 'TestWatchImageRef|TestServiceWatchesImage|TestParseManifestDigest|TestPickRepoDigest|TestCheckImageUpdates|TestImageCheckInterval|TestFetchImageRepoDigest|TestNoopRuntime_AllOperations|TestParseCompose'`
- `go test ./internal/server/ -run 'TestDataEventFromImageUpdate|TestPublishHomelabImageUpdates'`
- `go test ./internal/store/ -run TestEventStore_DataEventExists`
- `go test ./pkg/config/ -run TestValidate`
- `go test ./pkg/pipeline/ -run TestLoadBuiltinBlueprints`
- BDD (Docker): `tests/specs/event_spec_test.go` (Homelab image update events), `tests/specs/blueprint_page_spec_test.go` (`homelab_image_update_notify`)
- User guide: [homelab-image-updates.md](../../../docs/user-guide/homelab-image-updates.md)
