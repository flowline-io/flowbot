# Function publish version starts at v1

Status: implemented

## Problem

First publish of a named function showed **Latest published v2** (and versioned call path `/v/2`) because `PublishDefinition` stored the snapshot at `draft_revision + 1`. Draft revision already starts at 1 and bumps on each save/publish for optimistic locking, so published semver was coupled to draft edits instead of counting publish events.

## Fix

- `FunctionStore.PublishDefinition` assigns snapshot `version` as `max(existing published snapshot) + 1` (or 1 when none).
- Draft `function_definitions.version` still bumps on publish for optimistic locking only.
- `Service.ApplyBundle` returns the latest published snapshot version (same as `Publish`), not the post-publish draft revision.

## Notes

- Existing deployments may already have snapshots at v2+; new publishes continue from their max snapshot version.
- Pipeline versioning is unchanged (separate store path).
- Versioned call URL on the editor uses Alpine `callVersionURL()` (derived from `data-call-url` + `publishedVersion`) so the link appears immediately after client-side publish without reload.
